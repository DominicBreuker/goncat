#!/bin/sh

transport="$1"
shift

echo 'Testing...' 
fails=0
for current_test in "$@"; do
  echo ''
  echo "##### -> Current test: $current_test with transport=$transport <- #####"
  echo ''

  if $current_test "$transport"; then
    echo "Client Success: $current_test"
  else
    echo "Client Failure: $current_test"
    fails=$(( $fails + 1 ))
  fi

  echo ''
  echo '####################################################'
done

echo "Fails: $fails"
if [[ "$fails" -gt 0 ]]; then
  echo 'FAIL'
  exit 1
fi

echo 'SUCCESS'
exit 0
