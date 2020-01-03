package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestCreateDamageVerifyRestore(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	dataFilename, err := createTestInput()
	if err != nil {
		t.Errorf("Error creating tempfile: %s", err.Error())
	}
	createPresFile(dataFilename)
	presFilename := fmt.Sprint(dataFilename, ".pres")
	srcFilename := fmt.Sprint(dataFilename, ".orig")
	err = os.Rename(dataFilename, srcFilename)
	if err != nil {
		t.Errorf("Error renaming file: %s", err.Error())
	}
	err = damageOneByte(presFilename)
	if err != nil {
		t.Errorf("Error renaming file: %s", err.Error())
	}
	verifyPresFile(presFilename)
	restoreData(presFilename)
	eq, err := filesAreEqual(srcFilename, dataFilename)
	if err != nil {
		t.Errorf("Error comparing files: %s", err.Error())
	}
	if !eq {
		t.Errorf("Restored data does not match the original")
	}
	if err := os.Remove(dataFilename); err != nil {
		t.Errorf("Error removing tempfile: %s", err.Error())
	}
	if err := os.Remove(srcFilename); err != nil {
		t.Errorf("Error removing tempfile: %s", err.Error())
	}
	if err := os.Remove(presFilename); err != nil {
		t.Errorf("Error removing tempfile: %s", err.Error())
	}
}

func createTestInput() (string, error) {
	fileSize := (dataShardCnt * dataShardCnt) + (rand.Int() % 22e3)
	content := make([]byte, fileSize)
	_, err := rand.Read(content)
	if err != nil {
		return "", err
	}
	tempFile, err := ioutil.TempFile("", "pres_test_input_*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	_, err = tempFile.Write(content)
	return tempFile.Name(), err
}

func damageOneByte(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	damageIndex := rand.Int() % len(content)
	damageByte := make([]byte, 1)
	_, err = rand.Read(damageByte)
	if err != nil {
		return err
	}
	content[damageIndex] = damageByte[0]
	return ioutil.WriteFile(filename, content, 0644)
}

func filesAreEqual(a, b string) (bool, error) {
	contentA, err := ioutil.ReadFile(a)
	if err != nil {
		return false, err
	}
	contentB, err := ioutil.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(contentA, contentB), nil
}
