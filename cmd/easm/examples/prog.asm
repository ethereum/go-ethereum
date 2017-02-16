	jump @main
my_label:
	push 0
	push 1
	mstore

	push 0
	push 5
	push 10
	log1
	stop
main:
	push 1
	push 1
	eq

	jumpi @my_label
