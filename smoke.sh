DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
SERVICED=${DIR}/serviced/serviced
IP=$(/sbin/ifconfig eth0 | grep 'inet addr:' | cut -d: -f2 | awk {'print $1'})
HOSTNAME=$(hostname)

succeed() {
    echo ===== SUCCESS =====
    echo $@
    echo ===================
}

fail() {
    echo ====== FAIL ======
    echo $@
    echo ==================
    exit 1
}

# Add the vhost to /etc/hosts so we can resolve it for the test
add_to_etc_hosts() {
    if [ -z "$(grep -e "^${IP} websvc.${HOSTNAME}" /etc/hosts)" ]; then
        sudo /bin/bash -c "echo ${IP} websvc.${HOSTNAME} >> /etc/hosts"
    fi
}

cleanup() {
    echo "Killing serviced at pid ${SERVICED_PID}..."
    sudo kill -9 ${SERVICED_PID}
    echo "Killing extant containers..."
    docker kill $(docker ps -q)
    sleep 1
    echo "Clearing out serviced state..."
    sudo rm -rf /tmp/serviced-root/var/isvcs/*
    echo
}
trap cleanup EXIT


start_serviced() {
    echo "Starting serviced..."
    sudo GOPATH=${GOPATH} PATH=${PATH} ${PWD}/serviced/serviced -master -agent &
    SERVICED_PID=$[$!+3]
    echo "Waiting 60 seconds for serviced to become the leader..."
    for i in {1..60}; do
        wget --no-check-certificate http://${HOSTNAME}:443 &>/dev/null && return 0
        sleep 1
    done
    return 1
}

# Add a host
add_host() {
    HOST_ID=$(${SERVICED} host add "${IP}:4979" default)
    sleep 1
    [ -z "$(${SERVICED} host list ${HOST_ID} 2>/dev/null)" ] && return 1
    return 0
}

add_template() {
    TEMPLATE_ID=$(${SERVICED} template compile ${DIR}/dao/testsvc | ${SERVICED} template add)
    sleep 1
    [ -z "$(${SERVICED} template list ${TEMPLATE_ID})" ] && return 1
    return 0
}

deploy_service() {
    echo "Deploying template id ${TEMPLATE_ID}"
    echo ${SERVICED} service deploy ${TEMPLATE_ID} default testsvc
    SERVICE_ID=$(${SERVICED} template deploy ${TEMPLATE_ID} default testsvc)
    sleep 2
    [ -z "$(${SERVICED} service list ${SERVICE_ID})" ] && return 1
    return 0
}

start_service() {
    ${SERVICED} service start ${SERVICE_ID}
    sleep 10 
    [ -z "${SERVICED} service list ${SERVICE_ID}" ] && return 1
    return 0
}

test_vhost() {
    wget --no-check-certificate -qO- https://websvc.${HOSTNAME} &>/dev/null || return 1
    return 0
}

test_assigned_ip() {
    wget ${IP}:1000 -qO- &>/dev/null || return 1
    return 0
}

test_config() {
    wget --no-check-certificate -qO- ${IP}:1000/etc/my.cnf | grep "innodb_buffer_pool_size"  || return 1
    return 0
}

test_dir_config() {
    [ "$(wget --no-check-certificate -qO- https://websvc.${HOSTNAME}/etc/bar.txt)" == "baz" ] || return 1
    return 0
}

install_wget() {
    ${SERVICED} service attach s1 apt-get -y install wget
}

test_port_mapped() {
    [ "$(${SERVICED} service attach s1 wget -qO- http://localhost:9090/etc/bar.txt)" == "baz" ] || return 1
    return 0
}

# Setup
add_to_etc_hosts

# Run all the tests
start_serviced    && succeed "Serviced became leader within timeout"    || fail "serviced failed to become the leader within 60 seconds."
add_host          && succeed "Added host successfully"                  || fail "Unable to add host"
add_template      && succeed "Added template successfully"              || fail "Unable to add template"
deploy_service    && succeed "Deployed service successfully"            || fail "Unable to deploy service"
start_service     && succeed "Started service"                          || fail "Unable to start service"
test_vhost        && succeed "VHost is up and listening"                || fail "Unable to access service VHost"
test_assigned_ip  && succeed "Assigned IP is listening"                 || fail "Unable to access service by assigned IP"
test_config       && succeed "Config file was successfully injected"    || fail "Unable to access config file"
test_dir_config   && succeed "-CONFIGS- file was successfully injected" || fail "Unable to access -CONFIGS- file"
install_wget
test_port_mapped  && succeed "Port is imported correctly"               || fail "Port was not imported correctly"

# "trap cleanup EXIT", above, will handle cleanup
