// Copyright 2014 The go-amazoncoin Authors
// This file is part of the go-amazoncoin library.
//
// The go-amazoncoin library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-amazoncoin library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-amazoncoin library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"

	"github.com/amazoncoinio/go-amazoncoin/common"
	"github.com/amazoncoinio/go-amazoncoin/rlp"
	"github.com/amazoncoinio/go-amazoncoin/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) common.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie.Hash()
}
