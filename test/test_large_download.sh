#!bin/bash

# Test the download script
go run ../main.go large_test.torrent

FILE_HASH=$(sha256sum debian-13.3.0-amd64-netinst.iso | awk '{print $1}')

# This hash was taken directly from the debian website at https://cdimage.debian.org/debian-cd/current/amd64/bt-cd/SHA256SUMS
ACTUAL_HASH="c9f09d24b7e834e6834f2ffa565b33d6f1f540d04bd25c79ad9953bc79a8ac02"

echo "################################### TEST RESULTS ###################################"

if [ "$FILE_HASH" = "$ACTUAL_HASH" ]; then
    echo "Test passed: File hash matches expected value."
else
    echo "Test failed: File hash does not match expected value."
    echo "Expected: $ACTUAL_HASH"
    echo "Actual:   $FILE_HASH"
fi


# Cleanup
rm debian-13.3.0-amd64-netinst.iso