// Copyright 2025 PT. Fajar Mitra Indah. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"site-dev-backend/cmd"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		logrus.Error(err.Error())
		os.Exit(1)
	}
}
