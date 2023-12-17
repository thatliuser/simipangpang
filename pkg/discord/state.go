// Saving / loading / backup of bot state to disk.

package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

func (b *Bot) Load() error {
	contents, err := os.ReadFile(saveFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("couldn't read contents of file %v: %v", saveFile, err)
		} else {
			return nil
		}
	}
	if err := json.Unmarshal(contents, &b.servers); err != nil {
		return fmt.Errorf("couldn't unmarshal contents: %v", err)
	}
	return nil
}

// Back up the save file
func (b *Bot) Backup() error {
	contents, err := os.ReadFile(saveFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			return fmt.Errorf("couldn't read contents of file %v: %v", saveFile, err)
		}
	}

	filename := fmt.Sprintf("%v-%v%v", saveName, time.Now().Unix(), saveExt)
	backup, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("couldn't create file %v: %v", filename, err)
	}
	defer backup.Close()

	if _, err := backup.Write(contents); err != nil {
		return fmt.Errorf("couldn't write contents to backup file %v: %v", filename, err)
	}

	return nil
}

func (b *Bot) Save() error {
	// Backup the existing file if it exists
	if err := b.Backup(); err != nil {
		return fmt.Errorf("couldn't copy contents to backup: %v", err)
	}

	// Serialize first (if the data fails to marshal we don't want to truncate the file)
	data, err := json.MarshalIndent(b.servers, "", "\t")
	if err != nil {
		return fmt.Errorf("couldn't marshal server data: %v", err)
	}
	// Open the save file
	file, err := os.Create(saveFile)
	if err != nil {
		return fmt.Errorf("couldn't create savefile %v: %v", saveFile, err)
	}
	defer file.Close()
	// Write the serialized data to the file
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("couldn't write data to savefile %v: %v", saveFile, err)
	}

	return nil
}
