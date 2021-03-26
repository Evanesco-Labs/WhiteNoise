/*
 * Copyright (C) 2019 The themis Authors
 * This file is part of The themis library.
 *
 * The themis is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The themis is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The themis.  If not, see <http://www.gnu.org/licenses/>.
 */
package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func logPrint() {
	Trace("trace")
	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	Fatal("fatal")

	testValue := 1
	Tracef("trace %v", testValue)
	Debugf("debug %v", testValue)
	Infof("info %v", testValue)
	Warnf("warn %v", testValue)
	Errorf("error %v", testValue)
	Fatalf("fatal %v", testValue)

	fmt.Println("=========")
}

func TestLog(t *testing.T) {
	defer func() {
		os.RemoveAll("Log/")
	}()

	InitLog(TraceLog, Stdout)
	Log.SetDebugLevel(TraceLog)
	logPrint()

	Log.SetDebugLevel(FatalLog)
	SetModuleLevel("log", InfoLog)
	logPrint()

	Log.SetDebugLevel(TraceLog)
	SetModuleLevel("log", FatalLog)
	logPrint()
	err := ClosePrintLog()
	assert.Nil(t, err)
}

func TestNewLogFile(t *testing.T) {
	defer func() {
		os.RemoveAll("Log/")
	}()
	Init(PATH, Stdout)
	logfileNum1, err1 := ioutil.ReadDir("Log/")
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	logPrint()
	isNeedNewFile := CheckIfNeedNewFile()
	assert.NotEqual(t, isNeedNewFile, true)
	ClosePrintLog()
	time.Sleep(time.Second * 2)
	Init(PATH, Stdout)
	logfileNum2, err2 := ioutil.ReadDir("Log/")
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	assert.Equal(t, len(logfileNum1), (len(logfileNum2) - 1))
}
