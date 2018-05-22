cd ./shyftDb/postgres_setup
sh drop_tables.sh && sh init_tables.sh
cd ..
cd ..
sh resetShyftGeth.sh && sh initShyftGeth.sh