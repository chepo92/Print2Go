package webapi

import (
	"debug/elf"
	"fmt"
	"os"
)

func validateUpdateBinary(path string) error {

	file, err := elf.Open(path)

	if err != nil {
		return fmt.Errorf("invalid ELF binary")
	}

	defer file.Close()

	if file.Machine != elf.EM_MIPS {
		return fmt.Errorf(
			"invalid architecture: %v",
			file.Machine,
		)
	}

	return nil
}

func createUpdateScript() error {

	const script = `#!/bin/sh

sleep 2

if [ ! -f /tmp/Print2Go.new ]; then
	exit 1
fi


/etc/init.d/Print2Go-srv stop

sleep 1


if [ -f /usr/bin/Print2Go ]; then
	cp /usr/bin/Print2Go /usr/bin/Print2Go.previous
fi


cp /tmp/Print2Go.new /usr/bin/Print2Go

if [ $? -ne 0 ]; then

	cp /usr/bin/Print2Go.previous /usr/bin/Print2Go

	chmod +x /usr/bin/Print2Go

	/etc/init.d/Print2Go-srv start

	exit 1
fi


sync

chmod +x /usr/bin/Print2Go


/etc/init.d/Print2Go-srv start


rm -f /tmp/print2go-update.sh
rm -f /tmp/Print2Go.new	
`

	err := os.WriteFile(
		updateScriptPath,
		[]byte(script),
		0755,
	)

	return err
}
