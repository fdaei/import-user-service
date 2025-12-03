package user

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
)


func streamUsers(ctx context.Context, reader io.Reader, out chan<- ImportUser) (int, error) {
	br := bufio.NewReader(reader)
	first, err := peekFirstNonSpace(br)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil
		}
		return 0, err
	}

	dec := json.NewDecoder(br)
	dec.UseNumber()
	total := 0

	if first == '[' {
		if _, err := dec.Token(); err != nil { // Consume '['
			return 0, err
		}
		for dec.More() {
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			var u ImportUser
			if err := dec.Decode(&u); err != nil {
				return total, err
			}
			total++
			out <- u
		}
		if _, err := dec.Token(); err != nil { // Consume ']'
			return total, err
		}
		return total, nil
	}

	for {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		var u ImportUser
		if err := dec.Decode(&u); err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
		total++
		out <- u
	}
}

func peekFirstNonSpace(br *bufio.Reader) (byte, error) {
	for {
		b, err := br.Peek(1)
		if err != nil {
			return 0, err
		}
		if len(b) == 0 {
			return 0, io.EOF
		}

		switch b[0] {
		case ' ', '\n', '\r', '\t':
			if _, err := br.ReadByte(); err != nil {
				return 0, err
			}
			continue
		}
		return b[0], nil
	}
}
