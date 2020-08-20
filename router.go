// Copyright (c) 2015, Marios Andreopoulos.
//
// This file is part of aislib.
//
//  Aislib is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
//  Aislib is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
// along with aislib.  If not, see <http://www.gnu.org/licenses/>.

package aislib

import (
	"errors"
	"strconv"
	"strings"
)

// A Message stores the important properties of a AIS message, including only information useful
// for decoding: Type, Payload, Padding Bits
// A Message should come after processing one or more AIS radio sentences (checksum check,
// concatenate payloads spanning across sentences, etc).
type Message struct {
	Type    uint8
	Payload string
	Padding uint8
}

// FailedSentence includes an AIS sentence that failed to process (e.g wrong checksum) and the reason
// it failed.
type FailedSentence struct {
	Sentence string
	Issue    string
}

// Router accepts AIS radio sentences and process them. It checks their checksum,
// and AIS identifiers. If they are valid it tries to assemble the payload if it spans
// on multiple sentences. Upon success it returns the AIS Message at the out channel.
// Failed sentences go to the err channel.
// If the in channel is closed, then it sends a message with type 255 at the out channel.
// Your function can check for this message to know when it is safe to exit the program.
func Router(sentence string) (*Message, error) {
	count, ccount, padding := 0, 0, 0
	size, id := "0", "0"
	payload := ""
	var cache [5]string
	var err error
	aisIdentifiers := map[string]bool{
		"ABVD": true, "ADVD": true, "AIVD": true, "ANVD": true, "ARVD": true,
		"ASVD": true, "ATVD": true, "AXVD": true, "BSVD": true, "SAVD": true,
	}
	if len(sentence) == 0 { // Do not process empty lines
		return nil, errors.New("empty line")
	}
	tokens := strings.Split(sentence, ",") // I think this takes the major portion of time for this function (after benchmarking)

	if !Nmea183ChecksumCheck(sentence) { // Checksum check
		return nil, errors.New("checksum failed")
	}

	if !aisIdentifiers[tokens[0][1:5]] { // Check for valid AIS identifier
		return nil, errors.New("sentence isn't AIVDM/AIVDO")
	}

	if tokens[1] == "1" { // One sentence message, process it immediately
		return &Message{MessageType(tokens[5]), tokens[5], uint8(padding)}, nil
	} else { // Message spans across sentences.
		ccount, err = strconv.Atoi(tokens[2])
		if err != nil {
			return nil, errors.New("here: " + tokens[2])
		}
		if ccount != count+1 || // If there are sentences with wrong seq.number in cache send them as failed
			(tokens[3] != id && count != 0) || // If there are sentences with different sequence id in cache , send old parts as failed
			(tokens[1] != size && count != 0) { // If there messages with wrong size in cache, send them as failed
			for i := 0; i < count; i++ {
				return nil, errors.New("incomplete/out of order span sentence")
			}
			if ccount != 1 { // The current one is invalid too
				return nil, errors.New("incomplete/out of order span sentence")
			}
			count = 0
			payload = ""
		}
		payload += tokens[5]
		cache[ccount-1] = sentence
		count++
		if ccount == 1 { // First message in sequence, get size and id
			size = tokens[1]
			id = tokens[3]
		} else if size == tokens[2] && count == ccount { // Last message in sequence, send it and clean up.
			padding, _ = strconv.Atoi(tokens[6][:1])
			count = 0
			payload = ""
			return &Message{MessageType(payload), payload, uint8(padding)}, nil
		}
	}
	return &Message{255, "", 0}, nil
}
