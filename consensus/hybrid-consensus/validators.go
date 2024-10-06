package hybridconsensus

// Validator ekleme, çıkarma ve doğrulama işlevleri
func (hc *HybridConsensus) AddValidator(address string, stake uint64) {
    hc.validators = append(hc.validators, Validator{Address: address, Stake: stake})
}

func (hc *HybridConsensus) RemoveValidator(address string) {
    for i, validator := range hc.validators {
        if validator.Address == address {
            hc.validators = append(hc.validators[:i], hc.validators[i+1:]...)
            return
        }
    }
}

