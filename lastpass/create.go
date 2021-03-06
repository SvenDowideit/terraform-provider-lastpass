package lastpass

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Create is used to create a new resource and generate ID.
func (c *Client) Create(s Secret) (Secret, error) {
	err := c.login()
	if err != nil {
		return s, err
	}
	template := s.getTemplate()
	cmd := exec.Command("lpass", "add", s.Name, "--non-interactive", "--sync=now")
	var inbuf, errbuf bytes.Buffer
	inbuf.Write([]byte(template))
	cmd.Stdin = &inbuf
	cmd.Stderr = &errbuf
	err = cmd.Run()
	if err != nil {
		var err = errors.New(errbuf.String())
		return s, err
	}
	var outbuf bytes.Buffer
	var secrets []Secret
	// because of the ridiculous way lpass sync works we will need to retry until we get our ID.
	// see open issue at https://github.com/lastpass/lastpass-cli/issues/450
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 2)
		errbuf.Reset()
		outbuf.Reset()
		cmd = exec.Command("lpass", "sync")
		cmd.Stderr = &errbuf
		err = cmd.Run()
		if err != nil {
			var err = errors.New(errbuf.String())
			return s, err
		}
		cmd = exec.Command("lpass", "show", "--sync=now", s.Name, "--json", "-x")
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		err = cmd.Run()
		if err != nil {
			if !strings.Contains(errbuf.String(), "Could not find specified account") {
				var err = errors.New(errbuf.String())
				return s, err
			}
			continue
		}
		err = json.Unmarshal(outbuf.Bytes(), &secrets)
		if err != nil {
			return s, err
		}
		if len(secrets) > 1 {
			err := errors.New("more than one secret with same name, unable to determine ID")
			return s, err
		}
		if secrets[0].ID == "0" {
			// sync is still not done with upstream.
			continue
		}
		return secrets[0], nil
	}
	err = errors.New("timeout, unable to create new secret")
	return s, err
}
