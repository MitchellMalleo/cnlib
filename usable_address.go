package cnlib

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

/// Type Definition

// usableAddress is a wrapper struct that can provide a usable output address.
type usableAddress struct {
	Wallet            *HDWallet
	DerivationPath    *DerivationPath
	derivedPrivateKey *btcec.PrivateKey // derived from master along a derivation path, or specific pk from sweep.
}

/// Constructors

// newUsableAddressWithDerivationPath accepts a wallet and derivation path, and returns a pointer to a UsableAddress.
func newUsableAddressWithDerivationPath(wallet *HDWallet, derivationPath *DerivationPath) (*usableAddress, error) {
	kf := keyFactory{masterPrivateKey: wallet.masterPrivateKey}

	indexKey, err := kf.indexPrivateKey(derivationPath)
	if err != nil {
		return nil, err
	}

	ecPriv, err := indexKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	ua := usableAddress{Wallet: wallet, DerivationPath: derivationPath, derivedPrivateKey: ecPriv}
	return &ua, nil
}

// newUsableAddressWithImportedPrivateKey accepts a wallet and imported private key, and returns a pointer to a UsableAddress.
func newUsableAddressWithImportedPrivateKey(wallet *HDWallet, importedPrivateKey *ImportedPrivateKey) *usableAddress {
	ecPriv := importedPrivateKey.wif.PrivKey
	ua := usableAddress{Wallet: wallet, DerivationPath: nil, derivedPrivateKey: ecPriv}
	return &ua
}

/// Receiver methods

// MetaAddress returns a meta address with a given path based on wallet's BaseCoin, and uncompressed pubkey if a receive address. usableAddress's DerivationPath must not be nil.
func (ua *usableAddress) MetaAddress() (*MetaAddress, error) {
	addr, err := ua.generateAddress()

	if err != nil {
		return nil, err
	}

	path := ua.DerivationPath
	if path == nil {
		return nil, errors.New("found nil derivation path")
	}

	ecPub := ua.derivedPrivateKey.PubKey()
	pubkeyBytes := ecPub.SerializeUncompressed()
	pubkey := ""
	if path.Change == 0 {
		pubkey = hex.EncodeToString(pubkeyBytes)
	}

	ma := MetaAddress{Address: addr, DerivationPath: path, UncompressedPublicKey: pubkey}
	return &ma, nil
}

// BIP49AddressFromPubkeyHash returns a P2SH-P2WPKH address from a pubkey's Hash160.
func bip49AddressFromPubkeyHash(hash []byte, basecoin *BaseCoin) (string, error) {
	scriptSig, err := txscript.NewScriptBuilder().AddOp(txscript.OP_0).AddData(hash).Script()
	if err != nil {
		return "", err
	}
	addrHash, err := btcutil.NewAddressScriptHash(scriptSig, basecoin.defaultNetParams())
	if err != nil {
		return "", err
	}
	return addrHash.EncodeAddress(), nil
}

// BIP84AddressFromPubkeyHash returns a native P2WPKH address from a pubkey's Hash160.
func bip84AddressFromPubkeyHash(hash []byte, basecoin *BaseCoin) (string, error) {
	addrHash, err := btcutil.NewAddressWitnessPubKeyHash(hash, basecoin.defaultNetParams())
	if err != nil {
		return "", err
	}
	return addrHash.EncodeAddress(), nil
}

/// Unexposed methods

func (ua *usableAddress) generateAddress() (string, error) {
	purpose := ua.DerivationPath.Purpose

	if purpose == bip84purpose {
		return ua.buildSegwitAddress(ua.DerivationPath)
	} else if purpose == bip49purpose {
		return ua.buildBIP49Address(ua.DerivationPath)
	}
	return "", errors.New("Unrecognized Address Purpose")
}

func (ua *usableAddress) buildBIP49Address(path *DerivationPath) (string, error) {
	ecPub := ua.derivedPrivateKey.PubKey()
	pubkeyBytes := ecPub.SerializeCompressed()
	keyHash := btcutil.Hash160(pubkeyBytes)
	return bip49AddressFromPubkeyHash(keyHash, ua.Wallet.BaseCoin)
}

func (ua *usableAddress) buildSegwitAddress(path *DerivationPath) (string, error) {
	ecPub := ua.derivedPrivateKey.PubKey()
	pubkeyBytes := ecPub.SerializeCompressed()
	keyHash := btcutil.Hash160(pubkeyBytes)
	return bip84AddressFromPubkeyHash(keyHash, ua.Wallet.BaseCoin)
}
