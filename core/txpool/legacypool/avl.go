package legacypool

import (
	"errors"
	"math/big"
)

var (
	ErrEmptyTree = errors.New("empty tree")
)

// AVLTree structure. Public methods are Add, Remove, Update, Search, Flatten.
type AVLTree struct {
	root *AVLNode
}

func (t *AVLTree) Add(key uint64, value *big.Int) {
	t.root = t.root.add(key, value)
}

func (t *AVLTree) Remove(key uint64) {
	t.root = t.root.remove(key)
}

func (t *AVLTree) Update(oldKey uint64, newKey uint64, newValue *big.Int) {
	t.root = t.root.remove(oldKey)
	t.root = t.root.add(newKey, newValue)
}

func (t *AVLTree) Search(key uint64) (node *AVLNode, sum *big.Int) {
	return t.root.search(key)
}

func (t *AVLTree) Smallest() (uint64, error) {
	if t.root == nil {
		return 0, ErrEmptyTree
	}
	return t.root.findSmallest().key, nil // might get error if root is nil
}

func (t *AVLTree) Largest() (uint64, error) {
	if t.root == nil {
		return 0, ErrEmptyTree
	}
	return t.root.findLargest().key, nil // might get error if root is nil
}

func (t *AVLTree) Flatten() []*AVLNode {
	nodes := make([]*AVLNode, 0)
	if t.root == nil {
		return nodes
	}
	t.root.displayNodesInOrder(&nodes)
	return nodes
}

// AVLNode structure
type AVLNode struct {
	key   uint64   // nonce
	value *big.Int // cost
	sum   *big.Int // Sum of costs of the subtree

	// height counts nodes (not edges)
	height int
	left   *AVLNode
	right  *AVLNode
}

// Adds a new node
func (n *AVLNode) add(key uint64, value *big.Int) *AVLNode {
	if n == nil {
		newValue := new(big.Int)
		newSum := new(big.Int)
		newValue.Add(newValue, value)
		newSum.Add(newSum, value)
		return &AVLNode{key, newValue, newSum, 1, nil, nil}
	}

	if key < n.key {
		n.left = n.left.add(key, value)

	} else if key > n.key {
		n.right = n.right.add(key, value)
	} else {
		// if same key exists update value
		new := new(big.Int)
		n.value = new.Add(value, new)
	}

	new := new(big.Int)
	n.sum = new.Add(n.value, new)
	if n.left != nil {
		n.sum = n.sum.Add(n.sum, n.left.sum)
	}
	if n.right != nil {
		n.sum = n.sum.Add(n.sum, n.right.sum)
	}
	return n.rebalanceTree()
}

// Removes a node
func (n *AVLNode) remove(key uint64) *AVLNode {
	if n == nil {
		return nil
	}
	if key < n.key {
		n.left = n.left.remove(key)
	} else if key > n.key {
		n.right = n.right.remove(key)
	} else {
		if n.left != nil && n.right != nil {
			// node to delete found with both children;
			// replace values with smallest node of the right sub-tree
			rightMinNode := n.right.findSmallest()
			n.key = rightMinNode.key
			n.value = rightMinNode.value
			// delete smallest node that we replaced
			n.right = n.right.remove(rightMinNode.key)
		} else if n.left != nil {
			// node only has left child
			n = n.left
		} else if n.right != nil {
			// node only has right child
			n = n.right
		} else {
			// node has no children
			n = nil
			return n
		}
	}
	new := new(big.Int)
	n.sum = new.Add(n.value, new)
	if n.left != nil {
		n.sum = n.sum.Add(n.sum, n.left.sum)
	}
	if n.right != nil {
		n.sum = n.sum.Add(n.sum, n.right.sum)
	}
	return n.rebalanceTree()
}

// Searches for a node
func (n *AVLNode) search(key uint64) (*AVLNode, *big.Int) {
	if n == nil {
		return nil, big.NewInt(0)
	}
	if key < n.key {
		return n.left.search(key)
	} else if key > n.key {
		node, sum := n.right.search(key)
		if n.left != nil {
			sum = sum.Add(sum, n.left.sum)
		}
		sum = sum.Add(sum, n.value)
		return node, sum

	} else {
		new := new(big.Int)
		new.Add(new, n.value)
		if n.left != nil {
			return n, new.Add(new, n.left.sum)
		} else {
			return n, new
		}
	}
}

func (n *AVLNode) displayNodesInOrder(nodes *[]*AVLNode) {
	if n.left != nil {
		n.left.displayNodesInOrder(nodes)
	}
	(*nodes) = append((*nodes), n)
	if n.right != nil {
		n.right.displayNodesInOrder(nodes)
	}
}

func (n *AVLNode) getHeight() int {
	if n == nil {
		return 0
	}
	return n.height
}

func (n *AVLNode) recalculateHeight() {
	n.height = 1 + max(n.left.getHeight(), n.right.getHeight())
}

// Checks if node is balanced and rebalance
func (n *AVLNode) rebalanceTree() *AVLNode {
	if n == nil {
		return n
	}
	n.recalculateHeight()

	// check balance factor and rotateLeft if right-heavy and rotateRight if left-heavy
	balanceFactor := n.left.getHeight() - n.right.getHeight()
	if balanceFactor == -2 {
		// check if child is left-heavy and rotateRight first
		if n.right.left.getHeight() > n.right.right.getHeight() {
			n.right = n.right.rotateRight()
		}
		return n.rotateLeft()
	} else if balanceFactor == 2 {
		// check if child is right-heavy and rotateLeft first
		if n.left.right.getHeight() > n.left.left.getHeight() {
			n.left = n.left.rotateLeft()
		}
		return n.rotateRight()
	}
	return n
}

// Rotate nodes left to balance node
func (n *AVLNode) rotateLeft() *AVLNode {
	newRoot := n.right
	new := new(big.Int)
	temp := new.Add(new, n.sum)

	if n.right != nil {
		n.sum = n.sum.Sub(n.sum, n.right.sum)
	}

	n.right = newRoot.left

	if n.right != nil {
		n.sum = n.sum.Add(n.sum, n.right.sum)
	}

	newRoot.left = n
	newRoot.sum = temp
	n.recalculateHeight()
	newRoot.recalculateHeight()
	return newRoot
}

// Rotate nodes right to balance node
func (n *AVLNode) rotateRight() *AVLNode {
	newRoot := n.left
	new := new(big.Int)
	temp := new.Add(new, n.sum)

	if n.left != nil {
		n.sum = n.sum.Sub(n.sum, n.left.sum)
	}

	n.left = newRoot.right

	if n.left != nil {
		n.sum = n.sum.Add(n.sum, n.left.sum)
	}

	newRoot.right = n
	newRoot.sum = temp
	n.recalculateHeight()
	newRoot.recalculateHeight()
	return newRoot
}

// Finds the smallest child (based on the key) for the current node
func (n *AVLNode) findSmallest() *AVLNode {
	if n.left != nil {
		return n.left.findSmallest()
	} else {
		return n
	}
}

// Finds the largest child (based on the key) for the current node
func (n *AVLNode) findLargest() *AVLNode {
	if n.right != nil {
		return n.right.findLargest()
	} else {
		return n
	}
}

// Returns max number - TODO: std lib seemed to only have a method for floats!
func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
