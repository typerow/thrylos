package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// AddPeer adds a new peer to the node's list of peers if it is not already present. This function ensures that
// the node maintains an up-to-date list of peers with which it can communicate. Duplicate addresses are ignored
// to prevent redundancy.
func (node *Node) AddPeer(peerAddress string) {
	for _, peer := range node.Peers {
		if peer == peerAddress {
			return // Peer is already in the list, no need to add again.
		}
	}
	node.Peers = append(node.Peers, peerAddress) // Add new peer to the list.
}

// DiscoverPeers iterates over the node's current list of peers and requests their peer lists. This allows the node
// to discover new peers in the network dynamically. Discovered peers are added using the AddPeer method.
func (node *Node) DiscoverPeers() {
	for _, peer := range node.Peers {
		resp, err := http.Get(peer + "/peers")
		if err != nil {
			fmt.Println("Failed to discover peers:", err)
			continue
		}
		defer resp.Body.Close()

		var discoveredPeers []string
		json.NewDecoder(resp.Body).Decode(&discoveredPeers) // Decode the list of peers from the response.

		for _, discoveredPeer := range discoveredPeers {
			node.AddPeer(discoveredPeer) // Add each discovered peer.
		}
	}
}

// SyncWithPeer fetches the blockchain from a specified peer and updates the node's blockchain if the peer's
// blockchain is longer. This function is part of the node's mechanism for maintaining consensus on the blockchain
// state across the network.
func (node *Node) SyncWithPeer(peer string) {
	resp, err := http.Get(peer + "/blockchain")
	if err != nil {
		fmt.Println("Error fetching the blockchain:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	peerBlockchain := &Blockchain{}
	err = json.Unmarshal(body, peerBlockchain) // Unmarshal the fetched blockchain.
	if err != nil {
		fmt.Println("Error unmarshalling the blockchain:", err)
		return
	}

	// Update the node's blockchain if the peer's blockchain is longer.
	if len(peerBlockchain.Blocks) > len(node.Blockchain.Blocks) {
		node.Blockchain.Mu.Lock()
		node.Blockchain.Blocks = peerBlockchain.Blocks // Assume direct replacement for simplicity; real logic might be more complex.
		node.Blockchain.Mu.Unlock()
	}
}

// Broadcast serializes the node's blockchain and sends it to all known peers. This function supports the network's
// consistency by ensuring that all peers have the latest state of the blockchain.
func (node *Node) Broadcast() {
	data, err := json.Marshal(node.Blockchain) // Serialize the node's blockchain.
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, peer := range node.Peers {
		resp, err := http.Post(peer+"/blockchain", "application/json", bytes.NewBuffer(data)) // Broadcast the blockchain to each peer.
		if err != nil {
			fmt.Println("Failed to broadcast to peer:", peer)
			continue
		}
		resp.Body.Close() // Close the response body to prevent resource leaks.
	}
}
