.PHONY: clean libzkp

clean:
	rm -f *.a *.so
	cargo clean

libzkp:
	cargo build --release
	cp $(PWD)/target/release/libzkp.so $(PWD)/
	find $(PWD)/target | grep libzktrie.so | xargs -I{} cp {} $(PWD)/
