package ast

import (
	"fmt"
	"github.com/robertkrimen/otto/file"
)

// CommentPosition determines where the comment is in a given context
type CommentPosition int

const (
	_        CommentPosition = iota
	LEADING                  // Before the pertinent expression
	TRAILING                 // After the pertinent expression
	KEY                      // After a key or keyword
	COLON                    // After a colon in a field declaration
	FINAL                    // Final comments in a block, not belonging to a specific expression or the comment after a trailing , in an array or object literal
	TBD
)

// Comment contains the data of the comment
type Comment struct {
	Begin    file.Idx
	Text     string
	Position CommentPosition
}

// String returns a stringified version of the position
func (cp CommentPosition) String() string {
	switch cp {
	case LEADING:
		return "Leading"
	case TRAILING:
		return "Trailing"
	case KEY:
		return "Key"
	case COLON:
		return "Colon"
	case FINAL:
		return "Final"
	default:
		return "???"
	}
}

// String returns a stringified version of the comment
func (c Comment) String() string {
	return fmt.Sprintf("Comment: %v", c.Text)
}

// CommentMap is the data structure where all found comments are stored
type CommentMap map[Node][]*Comment

// AddComment adds a single comment to the map
func (cm CommentMap) AddComment(node Node, comment *Comment) {
	list := cm[node]
	list = append(list, comment)

	cm[node] = list
}

// AddComments adds a slice of comments, given a node and an updated position
func (cm CommentMap) AddComments(node Node, comments []*Comment, position CommentPosition) {
	for _, comment := range comments {
		comment.Position = position
		cm.AddComment(node, comment)
	}
}

// Size returns the size of the map
func (cm CommentMap) Size() int {
	size := 0
	for _, comments := range cm {
		size += len(comments)
	}

	return size
}

// MoveComments moves comments with a given position from a node to another
func (cm CommentMap) MoveComments(from, to Node, position CommentPosition) {
	for i, c := range cm[from] {
		if c.Position == position {
			cm.AddComment(to, c)

			// Remove the comment from the "from" slice
			cm[from][i] = cm[from][len(cm[from])-1]
			cm[from][len(cm[from])-1] = nil
			cm[from] = cm[from][:len(cm[from])-1]
		}
	}
}
