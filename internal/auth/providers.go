package auth

import "github.com/ad/domru/config"

// Config-backed providers

type ConfigProvider struct{ Cfg *config.Config }

func (p *ConfigProvider) GetToken() (string, error)   { return p.Cfg.Token, nil }
func (p *ConfigProvider) RefreshToken() error         { return nil } // will be wired later with real refresh logic
func (p *ConfigProvider) GetOperatorID() (int, error) { return p.Cfg.Operator, nil }
