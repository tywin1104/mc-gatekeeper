#!/bin/bash
 if [ ! -z ${API_HOST} ]; then
 cat <<END
 window.REACT_APP_API_HOST="${API_HOST}";
END
fi