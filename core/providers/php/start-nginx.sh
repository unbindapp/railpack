#!/bin/bash

set -e

PORT=${PORT:-80}

# Set the port in the nginx config
sed -i "s/80/$PORT/g" /etc/nginx/railpack.conf

# Set the storage permissions for Laravel
if [ "$IS_LARAVEL" = "true" ]; then
    chmod -R ugo+rw /app/storage
fi

echo "Starting Nginx on port $PORT"

# Start php-fpm and nginx
php-fpm --fpm-config /etc/php-fpm.conf & nginx -c /etc/nginx/railpack.conf
