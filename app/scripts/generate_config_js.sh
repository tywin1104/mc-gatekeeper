#!/bin/bash
 if [ ! -z ${RECPTCHA_SITEKEY} ]; then
 cat <<END
 window.RECPTCHA_SITEKEY="${RECPTCHA_SITEKEY}";
END
fi
