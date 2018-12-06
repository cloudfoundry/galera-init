#!/usr/bin/env bash

cat > /tmp/init.sql <<EOF
CREATE USER 'root'@'%';
GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION;
EOF

>&2 echo "Initializing mysqld data directory"
mysqld --initialize-insecure --disable-log-error --init-file=/tmp/init.sql

>&2 echo "Execing galera-init $*"

exec galera-init "$@"