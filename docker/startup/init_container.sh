#!/usr/bin/env sh

sed -i -e "s|/api/|$backendUri|g" /app/public/index.html

/app/adguardfilter