// Package random provides a feedable CSPRNG.
//
// CSPRNG used is fortuna: github.com/seehuhn/fortuna
// By default the CSPRNG is fed by two sources:
// - OS RNG
// - Entropy gathered by context switching
package random
