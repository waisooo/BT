#!bin/bash

# Test the download script with a small file
go run ../main.go small_test.torrent

FILE_HASH=$(sha256sum sample.txt | awk '{print $1}')

ACTUAL_HASH="034779f836050853f4d520fef986db353633d64025241c993bebc0aaf56c081c"

echo "################################### TEST RESULTS ###################################"

if [ "$FILE_HASH" = "$ACTUAL_HASH" ]; then
    echo "Test passed: File hash matches expected value."
else
    echo "Test failed: File hash does not match expected value."
    echo "Expected: $ACTUAL_HASH"
    echo "Actual:   $FILE_HASH"
fi

# Cleanup
rm sample.txt