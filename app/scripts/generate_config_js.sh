#!/bin/bash
 if [ ! -z ${RECAPTCHA_SITEKEY} ]; then
 cat <<END
 window.RECAPTCHA_SITEKEY="${RECAPTCHA_SITEKEY}";
END
fi
