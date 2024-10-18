package core

import (
    "fmt"
)

type Book struct {
    title  string
    author string
    year   int
}

func ExampleBooks() {
    var book1 Book
    var book2 Book

    book1.title = "The Great Gatsby"
    book1.author = "F. Scott Fitzgerald"
    book1.year = 1925

    book2.title = "To Kill a Mockingbird"
    book2.author = "Harper Lee"
    book2.year = 1960

    fmt.Println("Book 1:")
    fmt.Println("Title:", book1.title)
    fmt.Println("Author:", book1.author)
    fmt.Println("Year:", book1.year)

    fmt.Println("\nBook 2:")
    fmt.Println("Title:", book2.title)
    fmt.Println("Author:", book2.author)
    fmt.Println("Year:", book2.year)
}
