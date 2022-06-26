package toolkit

import (
	"context"
	"errors"
	"fmt"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
)

var (
	AlgoExplorerTestnet = "http://testnet-api.algonode.network"
	AlgoExplorerMainnet = "http://mainnet-api.algonode.network"

	AlgoExplorerIndexerTestnet = "http://testnet-idx.algonode.network"
	AlgoExplorerIndexerMainnet = "http://mainnet-idx.algonode.network"
)

var (
	ErrBankWallet = errors.New("engine: address belongs to a bank")
)

// Node ...
type Node struct {
	Address  string
	Children []*Node
	Parent   *Node
}

// Engine ...
type Engine struct {
	ac *algod.Client
	ai *indexer.Client

	ignoreList map[string]struct{}
	Trees      map[string]*Node
}

// NewEngine ...
func NewEngine() (*Engine, error) {
	address := AlgoExplorerMainnet
	indexerAddress := AlgoExplorerIndexerMainnet

	ac, err := algod.MakeClient(address, "")
	if err != nil {
		return nil, fmt.Errorf("new_account_manager: algod: make_client: %v", err)
	}

	ai, err := indexer.MakeClient(indexerAddress, "")
	if err != nil {
		return nil, fmt.Errorf("new_account_manager: indexer: make_client: %v", err)
	}

	e := &Engine{}
	e.ac = ac
	e.ai = ai
	e.ignoreList = make(map[string]struct{})
	e.Trees = make(map[string]*Node)

	return e, nil
}

// TopAddresses ...
func (e *Engine) TopAddresses(ctx context.Context, addresses map[string]struct{}) error {
	if len(addresses) == 0 {
		return nil
	}

	for address := range addresses {
		fmt.Println(address)
		v, ok := e.Trees[address]
		if !ok {
			v = &Node{Address: address, Children: []*Node{}}
		}

		if err := e.Travel(ctx, v); err != nil {
			return err
		}
	}

	return nil
}

// Travel ...
func (e *Engine) Travel(ctx context.Context, childNode *Node) error {
	// CHECK IF ALREADY DISCOVERED
	if _, ok := e.Trees[childNode.Address]; ok {
		return nil
	}

	e.Trees[childNode.Address] = childNode

	// FIND PARENT
	parent, err := e.FindParent(ctx, childNode.Address)
	if err != nil {
		fmt.Println("Address: ", parent, "Error: ", err)
		return nil
	}

	// CHECK IF ADDRESS SHOULD BE IGNORED
	if _, ok := e.ignoreList[parent]; ok {
		return nil
	}

	if err := e.validateDeadEnd(ctx, parent); err != nil {
		if err == ErrBankWallet {
			e.ignoreList[parent] = struct{}{}
			return nil
		}
		fmt.Println(err)
		return nil
	}

	parentNode, ok := e.Trees[parent]
	if !ok {
		parentNode = &Node{Address: parent, Children: []*Node{}}
	}
	parentNode.Children = append(parentNode.Children, childNode)
	childNode.Parent = parentNode

	return e.Travel(ctx, parentNode)
}

// FindParents ...
func (e *Engine) FindParents(ctx context.Context, address string) ([]string, error) {
	addressChain := []string{AddKnownName(address)}
	// CHECK IF DEADEND ADDRESS
	if err := e.validateDeadEnd(ctx, address); err != nil {
		if err == ErrBankWallet {
			err = nil
		}

		return addressChain, err
	}

	// FIND PARENT
	parent, err := e.FindParent(ctx, address)
	if err != nil {
		if parent != "" {
			addressChain = append(addressChain, parent)
		}
		return addressChain, err
	}

	parentChain, err := e.FindParents(ctx, parent)
	addressChain = append(addressChain, parentChain...)
	if err != nil {
		return addressChain, err
	}

	return addressChain, nil
}

// FindParent ...
func (e *Engine) FindParent(ctx context.Context, address string) (string, error) {
	var parent string
	var earliestRound uint64
	var nextToken string
	for {
		resp, err := e.ai.LookupAccountTransactions(address).Limit(1000).TxType("pay").NextToken(nextToken).Do(ctx)
		if err != nil {
			return parent, err
		}

		for _, txn := range resp.Transactions {
			if (parent == "") || (txn.ConfirmedRound < earliestRound) {
				earliestRound = txn.ConfirmedRound
				parent = txn.Sender
			}
		}

		nextToken = resp.NextToken
		if nextToken == "" {
			break
		}
	}

	return parent, nil
}

func (e *Engine) validateDeadEnd(ctx context.Context, address string) error {
	// LOOKUP ADDRESS DETAILS
	_, account, err := e.ai.LookupAccountByID(address).IncludeAll(true).Do(ctx)
	if err != nil {
		return err
	}
	// SKIP IF BANK WALLET
	if _, ok := Exchanges[address]; ok {
		return ErrBankWallet
	}
	if account.Amount >= uint64(500000*1000000) {
		return ErrBankWallet
	}

	return nil
}
