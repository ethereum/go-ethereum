// Copyright 2024 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	migrateFromFlag = &cli.StringFlag{
		Name:     "from",
		Usage:    "Source file-based keystore directory",
		Required: true,
	}
	migrateToFlag = &cli.StringFlag{
		Name:     "to",
		Usage:    "Destination database keystore path",
		Required: true,
	}
	migrateKeystoreCommand = &cli.Command{
		Action:    migrateKeystore,
		Name:      "migrate-keystore",
		Usage:     "Migrate keys from file-based keystore to database keystore",
		ArgsUsage: "",
		Flags: []cli.Flag{
			logLevelFlag,
			migrateFromFlag,
			migrateToFlag,
			utils.LightKDFFlag,
			acceptFlag,
		},
		Description: `
The migrate-keystore command migrates all keys from a file-based keystore
to a database-backed keystore for improved scalability.

Example:
  clef migrate-keystore --from ~/.ethereum/keystore --to ~/.ethereum/keystore.db

Note: This operation requires entering the password for each key to decrypt
and re-encrypt it into the database. The original keystore files are not
modified or deleted.
`,
	}
)

func init() {
	app.Commands = append(app.Commands, migrateKeystoreCommand)
}

func migrateKeystore(c *cli.Context) error {
	if err := initialize(c); err != nil {
		return err
	}

	fromPath := c.String(migrateFromFlag.Name)
	toPath := c.String(migrateToFlag.Name)
	lightKdf := c.Bool(utils.LightKDFFlag.Name)

	// Validate source path
	info, err := os.Stat(fromPath)
	if err != nil {
		return fmt.Errorf("source keystore not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source keystore must be a directory: %s", fromPath)
	}

	// Check if destination already exists
	if _, err := os.Stat(toPath); err == nil {
		return fmt.Errorf("destination already exists: %s (remove it first if you want to start fresh)", toPath)
	}

	// Set scrypt parameters
	var n, p int
	if lightKdf {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	} else {
		n, p = keystore.StandardScryptN, keystore.StandardScryptP
	}

	// Create source keystore (file-based)
	srcKs := keystore.NewKeyStore(fromPath, n, p)
	srcAccounts := srcKs.Accounts()

	if len(srcAccounts) == 0 {
		log.Info("No accounts found in source keystore", "path", fromPath)
		return nil
	}

	log.Info("Found accounts to migrate", "count", len(srcAccounts), "from", fromPath, "to", toPath)

	// Create destination keystore (database-backed)
	dstKs, err := keystore.NewDBKeyStore(toPath, n, p)
	if err != nil {
		return fmt.Errorf("failed to create destination keystore: %w", err)
	}
	defer dstKs.Close()

	// Migrate each account
	migrated := 0
	skipped := 0
	failed := 0

	for i, account := range srcAccounts {
		log.Info("Migrating account", "index", i+1, "total", len(srcAccounts), "address", account.Address.Hex())

		// Check if account already exists in destination
		if dstKs.HasAddress(account.Address) {
			log.Warn("Account already exists in destination, skipping", "address", account.Address.Hex())
			skipped++
			continue
		}

		// Get password for this account
		prompt := fmt.Sprintf("Enter password for account %s", account.Address.Hex())
		password := utils.GetPassPhrase(prompt, false)

		// Export from source
		keyJSON, err := srcKs.Export(account, password, password)
		if err != nil {
			log.Error("Failed to export account", "address", account.Address.Hex(), "err", err)
			failed++
			continue
		}

		// Import to destination
		_, err = dstKs.Import(keyJSON, password, password)
		if err != nil {
			log.Error("Failed to import account", "address", account.Address.Hex(), "err", err)
			failed++
			continue
		}

		log.Info("Successfully migrated account", "address", account.Address.Hex())
		migrated++
	}

	// Print summary
	fmt.Println()
	log.Info("Migration complete",
		"migrated", migrated,
		"skipped", skipped,
		"failed", failed,
		"total", len(srcAccounts))

	if failed > 0 {
		return fmt.Errorf("%d account(s) failed to migrate", failed)
	}

	// Verify migration
	dstAccounts := dstKs.Accounts()
	log.Info("Verification", "accounts_in_destination", len(dstAccounts))

	fmt.Println()
	fmt.Println("Migration successful!")
	fmt.Println()
	fmt.Printf("To use the database keystore, run clef with:\n")
	fmt.Printf("  clef --keystore-type db --keystore %s\n", toPath)
	fmt.Println()
	fmt.Println("Your original keystore files have NOT been modified.")
	fmt.Println("Once you verify everything works, you may optionally backup and remove the old keystore.")

	return nil
}

// MigrateKeystoreBatch migrates keys without prompting for passwords.
// This is useful for programmatic migration when passwords are known.
func MigrateKeystoreBatch(fromPath, toPath string, passwords map[string]string, lightKdf bool) error {
	// Set scrypt parameters
	var n, p int
	if lightKdf {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	} else {
		n, p = keystore.StandardScryptN, keystore.StandardScryptP
	}

	// Create source keystore
	srcKs := keystore.NewKeyStore(fromPath, n, p)
	srcAccounts := srcKs.Accounts()

	if len(srcAccounts) == 0 {
		return nil
	}

	// Create destination keystore
	dstKs, err := keystore.NewDBKeyStore(toPath, n, p)
	if err != nil {
		return fmt.Errorf("failed to create destination keystore: %w", err)
	}
	defer dstKs.Close()

	// Migrate each account
	for _, account := range srcAccounts {
		addrHex := account.Address.Hex()
		password, ok := passwords[addrHex]
		if !ok {
			// Try lowercase
			password, ok = passwords[filepath.Base(account.URL.Path)]
			if !ok {
				return fmt.Errorf("no password provided for account %s", addrHex)
			}
		}

		// Skip if already exists
		if dstKs.HasAddress(account.Address) {
			continue
		}

		// Export and import
		keyJSON, err := srcKs.Export(account, password, password)
		if err != nil {
			return fmt.Errorf("failed to export account %s: %w", addrHex, err)
		}

		_, err = dstKs.Import(keyJSON, password, password)
		if err != nil {
			return fmt.Errorf("failed to import account %s: %w", addrHex, err)
		}
	}

	return nil
}
