// Copyright 2016 The go-amazoncoin Authors
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

package ethclient

import "github.com/amazoncoinio/go-amazoncoin"

// Verify that Client implements the Amazoncoin interfaces.
var (
	_ = amazoncoin.ChainReader(&Client{})
	_ = amazoncoin.TransactionReader(&Client{})
	_ = amazoncoin.ChainStateReader(&Client{})
	_ = amazoncoin.ChainSyncReader(&Client{})
	_ = amazoncoin.ContractCaller(&Client{})
	_ = amazoncoin.GasEstimator(&Client{})
	_ = amazoncoin.GasPricer(&Client{})
	_ = amazoncoin.LogFilterer(&Client{})
	_ = amazoncoin.PendingStateReader(&Client{})
	// _ = amazoncoin.PendingStateEventer(&Client{})
	_ = amazoncoin.PendingContractCaller(&Client{})
)
