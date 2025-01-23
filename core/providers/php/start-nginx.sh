#!/bin/bash

PORT=${PORT:-80}

sed -i "s/80/$PORT/g" /etc/nginx/nginx.conf

echo "Starting Nginx on port $PORT"

php-fpm --fpm-config /etc/php-fpm.conf & nginx -c /etc/nginx/nginx.conf
