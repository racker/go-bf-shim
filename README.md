# To build

    export GOPATH=$(pwd)
    go install bfshim

# To run

    bin/bfshim -h

shows help, including all default values.

Run without `-h` to actually do something to the specified Blueflood instance.

# Known Bugs

It does not yet authenticate, so it can only be used in staging.

For some reason, Blueflood fails with a 400 Cannot Parse Payload error even though we're sending it valid JSON.  We have not found out why yet.

