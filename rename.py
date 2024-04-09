import glob
import os
from tqdm import tqdm

for filename in tqdm(glob.glob("./**", recursive=True)):
    if os.path.isdir(filename):
        continue

    if filename == "./rename.py":
        continue

    with open(filename, "rb") as f:
        content = f.read()

    newContent = content.replace(
        b"github.com/ethereum/go-ethereum",
        b"github.com/tenderly/binance-geth",
    )

    newContent = newContent.replace(
        b"github.com/tenderly/binance-geth/crypto/secp256k1",
        b"github.com/ethereum/go-ethereum/crypto/secp256k1",
    )

    if newContent != content:
        with open(filename, "wb") as f:
            f.write(newContent)
