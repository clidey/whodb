/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package sqlident renders database object names as quoted identifiers.
package sqlident

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// IsSimple reports whether an identifier can be passed through metadata APIs
// that do not accept an explicitly quoted identifier expression.
func IsSimple(identifier string) bool {
	if identifier == "" {
		return true
	}
	for _, r := range identifier {
		if r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ErrInvalidIdentifier indicates that a database object name cannot be safely rendered.
var ErrInvalidIdentifier = errors.New("invalid database identifier")

// QuoteStyle identifies the delimiter rules for an identifier.
type QuoteStyle uint8

const (
	// Unknown indicates that identifier quoting has not been configured.
	Unknown QuoteStyle = iota
	// DoubleQuote doubles embedded double quotes.
	DoubleQuote
	// Backtick doubles embedded backticks.
	Backtick
	// Bracket doubles embedded closing brackets.
	Bracket
	// DoubleQuoteStrict rejects embedded double quotes.
	DoubleQuoteStrict
	// BacktickStrict rejects embedded backticks.
	BacktickStrict
)

// Name is a database object name with an optional namespace component.
type Name struct {
	namespace string
	object    string
}

// New creates a database object name from raw, unquoted components.
func New(namespace, object string) (Name, error) {
	if object == "" {
		return Name{}, fmt.Errorf("%w: object name is empty", ErrInvalidIdentifier)
	}
	if strings.ContainsRune(namespace, '\x00') || strings.ContainsRune(object, '\x00') {
		return Name{}, fmt.Errorf("%w: identifier contains NUL", ErrInvalidIdentifier)
	}
	return Name{namespace: namespace, object: object}, nil
}

// SQL renders the name as one or two separately quoted identifier components.
func (n Name) SQL(style QuoteStyle) (string, error) {
	object, err := quote(style, n.object)
	if err != nil {
		return "", err
	}
	if n.namespace == "" {
		return object, nil
	}
	namespace, err := quote(style, n.namespace)
	if err != nil {
		return "", err
	}
	return namespace + "." + object, nil
}

func quote(style QuoteStyle, identifier string) (string, error) {
	var open, close string
	var rejectEmbedded bool
	switch style {
	case DoubleQuote:
		open, close = `"`, `"`
	case Backtick:
		open, close = "`", "`"
	case Bracket:
		open, close = "[", "]"
	case DoubleQuoteStrict:
		open, close, rejectEmbedded = `"`, `"`, true
	case BacktickStrict:
		open, close, rejectEmbedded = "`", "`", true
	default:
		return "", fmt.Errorf("%w: quote style is not configured", ErrInvalidIdentifier)
	}
	if rejectEmbedded && strings.Contains(identifier, close) {
		return "", fmt.Errorf("%w: identifier contains unsupported delimiter", ErrInvalidIdentifier)
	}
	escaped := strings.ReplaceAll(identifier, close, close+close)
	return open + escaped + close, nil
}
