*** Settings ***
Library      OperatingSystem
#Library      RequestsLibrary
#Library      SSHLibrary      timeout=60s
#Library      String

Resource     ../../../variables/${VARIABLES}_variables.robot
Resource     ../../../libraries/all_libs.robot
Resource    ../../../libraries/pretty_keywords.robot

Suite Setup       Testsuite Setup
Suite Teardown    Testsuite Teardown

*** Variables ***
${VARIABLES}=               common
${ENV}=                     common
${NAME_VPP1_TAP1}=          vpp1_tap1
${NAME_VPP2_TAP1}=          vpp2_tap1
${MAC_VPP1_TAP1}=           12:21:21:11:11:11
${MAC_VPP2_TAP1}=           22:21:21:22:22:22
${IP_VPP1_TAP1}=            fd30::1:a:0:0:1
${IP_VPP2_TAP1}=            fd31::1:a:0:0:1
${IP_LINUX_VPP1_TAP1}=      fd30::1:a:0:0:2
${IP_LINUX_VPP2_TAP1}=      fd31::1:a:0:0:2
${IP_VPP1_TAP1_NETWORK}=    fd30::1:0:0:0:0
${IP_VPP2_TAP1_NETWORK}=    fd31::1:0:0:0:0
${NAME_VPP1_MEMIF1}=        vpp1_memif1
${NAME_VPP2_MEMIF1}=        vpp2_memif1
${MAC_VPP1_MEMIF1}=         13:21:21:11:11:11
${MAC_VPP2_MEMIF1}=         23:21:21:22:22:22
${IP_VPP1_MEMIF1}=          fd33::1:a:0:0:1
${IP_VPP2_MEMIF1}=          fd33::1:a:0:0:2
${PREFIX}=                  64
${UP_STATE}=                up
${SYNC_SLEEP}=         10s
# wait for resync vpps after restart
${RESYNC_WAIT}=        50s

*** Test Cases ***
Configure Environment
    [Tags]    setup
    Configure Environment 1

Show Interfaces Before Setup
    vpp_term: Show Interfaces    agent_vpp_1
    vpp_term: Show Interfaces    agent_vpp_2
    Write To Machine    vpp_agent_ctl    vpp-agent-ctl ${AGENT_VPP_ETCD_CONF_PATH} -ps

Add VPP1_TAP1 Interface
    vpp_term: Interface Not Exists  node=agent_vpp_1    mac=${MAC_VPP1_TAP1}
    vpp_ctl: Put TAPv2 Interface With IP    node=agent_vpp_1    name=${NAME_VPP1_TAP1}    mac=${MAC_VPP1_TAP1}    ip=${IP_VPP1_TAP1}    prefix=${PREFIX}    host_if_name=linux_${NAME_VPP1_TAP1}
    linux: Set Host TAP Interface    node=agent_vpp_1    host_if_name=linux_${NAME_VPP1_TAP1}    ip=${IP_LINUX_VPP1_TAP1}    prefix=${PREFIX}

Check VPP1_TAP1 Interface Is Created
    ${interfaces}=       vat_term: Interfaces Dump    node=agent_vpp_1
    Log                  ${interfaces}
    vpp_term: Interface Is Created    node=agent_vpp_1    mac=${MAC_VPP1_TAP1}
    ${actual_state}=    vpp_term: Check TAPv2 interface State    agent_vpp_1    ${NAME_VPP1_TAP1}    mac=${MAC_VPP1_TAP1}    ipv6=${IP_VPP1_TAP1}/${PREFIX}    state=${UP_STATE}

Check Ping Between VPP1 and linux_VPP1_TAP1 Interface
    linux: Check Ping    node=agent_vpp_1    ip=${IP_VPP1_TAP1}
    vpp_term: Check Ping    node=agent_vpp_1    ip=${IP_LINUX_VPP1_TAP1}

Add VPP1_memif1 Interface
    vpp_term: Interface Not Exists    node=agent_vpp_1    mac=${MAC_VPP1_MEMIF1}
    vpp_ctl: Put Memif Interface With IP    node=agent_vpp_1    name=${NAME_VPP1_MEMIF1}    mac=${MAC_VPP1_MEMIF1}    master=true    id=1    ip=${IP_VPP1_MEMIF1}    prefix=24    socket=memif.sock
    vpp_term: Interface Is Created    node=agent_vpp_1    mac=${MAC_VPP1_MEMIF1}

Add VPP2_TAP1 Interface
    vpp_term: Interface Not Exists  node=agent_vpp_2    mac=${MAC_VPP2_TAP1}
    vpp_ctl: Put TAPv2 Interface With IP    node=agent_vpp_2    name=${NAME_VPP2_TAP1}    mac=${MAC_VPP2_TAP1}    ip=${IP_VPP2_TAP1}    prefix=${PREFIX}    host_if_name=linux_${NAME_VPP2_TAP1}
    linux: Set Host TAP Interface    node=agent_vpp_2    host_if_name=linux_${NAME_VPP2_TAP1}    ip=${IP_LINUX_VPP2_TAP1}    prefix=${PREFIX}

Check VPP2_TAP1 Interface Is Created
    ${interfaces}=       vat_term: Interfaces Dump    node=agent_vpp_1
    Log                  ${interfaces}
    vpp_term: Interface Is Created    node=agent_vpp_2    mac=${MAC_VPP2_TAP1}
    ${actual_state}=    vpp_term: Check TAPv2 interface State    agent_vpp_2    ${NAME_VPP2_TAP1}    mac=${MAC_VPP2_TAP1}    ipv6=${IP_VPP2_TAP1}/${PREFIX}    state=${UP_STATE}

Check Ping Between VPP2 And linux_VPP2_TAP1 Interface
    linux: Check Ping    node=agent_vpp_2    ip=${IP_VPP2_TAP1}
    vpp_term: Check Ping    node=agent_vpp_2    ip=${IP_LINUX_VPP2_TAP1}

Add VPP2_memif1 Interface
    vpp_term: Interface Not Exists    node=agent_vpp_2    mac=${MAC_VPP2_MEMIF1}
    vpp_ctl: Put Memif Interface With IP    node=agent_vpp_2    name=${NAME_VPP2_MEMIF1}    mac=${MAC_VPP2_MEMIF1}    master=false    id=1    ip=${IP_VPP2_MEMIF1}    prefix=24    socket=memif.sock
    vpp_term: Interface Is Created    node=agent_vpp_1    mac=${MAC_VPP1_MEMIF1}

Check Ping From VPP1 To VPP2_memif1
    vpp_term: Check Ping    node=agent_vpp_1    ip=${IP_VPP2_MEMIF1}

Check Ping From VPP2 To VPP1_memif1
    vpp_term: Check Ping    node=agent_vpp_2    ip=${IP_VPP1_MEMIF1}

Ping From VPP1 Linux To VPP2_TAP1 And LINUX_VPP2_TAP1 Should Not Pass
    ${status1}=    Run Keyword And Return Status    linux: Check Ping    node=agent_vpp_1    ip=${IP_VPP2_TAP1}
    ${status2}=    Run Keyword And Return Status    linux: Check Ping    node=agent_vpp_1    ip=${IP_LINUX_VPP2_TAP1}
    Should Be Equal As Strings    ${status1}    False
    Should Be Equal As Strings    ${status2}    False

Ping From VPP2 Linux To VPP1_TAP1 And LINUX_VPP1_TAP1 Should Not Pass
    ${status1}=    Run Keyword And Return Status    linux: Check Ping    node=agent_vpp_2    ip=${IP_VPP1_TAP1}
    ${status2}=    Run Keyword And Return Status    linux: Check Ping    node=agent_vpp_2    ip=${IP_LINUX_VPP1_TAP1}
    Should Be Equal As Strings    ${status1}    False
    Should Be Equal As Strings    ${status2}    False

Add Static Route From VPP1 Linux To VPP2
    linux: Add Route    node=agent_vpp_1    destination_ip=${IP_VPP2_TAP1_NETWORK}    prefix=${PREFIX}    next_hop_ip=${IP_VPP1_TAP1}

Add Static Route From VPP1 To VPP2
    Create Route On agent_vpp_1 With IP 20.20.1.0/24 With Next Hop 192.168.1.2 And Vrf Id 0

Add Static Route From VPP2 Linux To VPP1
    linux: Add Route    node=agent_vpp_2    destination_ip=${IP_VPP1_TAP1_NETWORK}    prefix=${PREFIX}    next_hop_ip=${IP_VPP2_TAP1}

Add Static Route From VPP2 To VPP1
    Create Route On agent_vpp_2 With IP 10.10.1.0/24 With Next Hop 192.168.1.1 And Vrf Id 0

Check Ping From VPP1 Linux To VPP2_TAP1 And LINUX_VPP2_TAP1
    linux: Check Ping    node=agent_vpp_1    ip=${IP_VPP2_TAP1}
    linux: Check Ping    node=agent_vpp_1    ip=${IP_LINUX_VPP2_TAP1}

Check Ping From VPP2 Linux To VPP1_TAP1 And LINUX_VPP1_TAP1
    linux: Check Ping    node=agent_vpp_2    ip=${IP_VPP1_TAP1}
    linux: Check Ping    node=agent_vpp_2    ip=${IP_LINUX_VPP1_TAP1}

Remove VPP Nodes
    Remove All Nodes
    Sleep    ${SYNC_SLEEP}

Start VPP1 And VPP2 Again
    Add Agent VPP Node    agent_vpp_1
    Add Agent VPP Node    agent_vpp_2
    Sleep    ${RESYNC_WAIT}

Create linux_VPP1_TAP1 And linux_VPP2_TAP1 Interfaces After Resync
    linux: Set Host TAP Interface    node=agent_vpp_1    host_if_name=linux_${NAME_VPP1_TAP1}    ip=${IP_LINUX_VPP1_TAP1}    prefix=${PREFIX}
    linux: Set Host TAP Interface    node=agent_vpp_2    host_if_name=linux_${NAME_VPP2_TAP1}    ip=${IP_LINUX_VPP2_TAP1}    prefix=${PREFIX}

Check Linux Interfaces On VPP1 After Resync
    ${out}=    Execute In Container    agent_vpp_1    ip a
    Log    ${out}
    Should Contain    ${out}    linux_${NAME_VPP1_TAP1}

Check Interfaces On VPP1 After Resync
    ${out}=    vpp_term: Show Interfaces    agent_vpp_1
    Log    ${out}
    ${int}=    vpp_ctl: Get Interface Internal Name    node=agent_vpp_1    interface=${NAME_VPP1_MEMIF1}
    Should Contain    ${out}    ${int}
    ${int}=    vpp_ctl: Get Interface Internal Name    node=agent_vpp_1    interface=${NAME_VPP1_TAP1}
    Should Contain    ${out}    ${int}

Check Linux Interfaces On VPP2 After Resync
    ${out}=    Execute In Container    agent_vpp_2    ip a
    Log    ${out}
    Should Contain    ${out}    linux_${NAME_VPP2_TAP1}

Check Interfaces On VPP2 After Resync
    ${out}=    vpp_term: Show Interfaces    agent_vpp_2
    Log    ${out}
    ${int}=    vpp_ctl: Get Interface Internal Name    node=agent_vpp_2    interface=${NAME_VPP2_MEMIF1}
    Should Contain    ${out}    ${int}
    ${int}=    vpp_ctl: Get Interface Internal Name    node=agent_vpp_2    interface=${NAME_VPP2_TAP1}
    Should Contain    ${out}    ${int}

Add Static Route From VPP1 Linux To VPP2 After Resync
    linux: Add Route    node=agent_vpp_1    destination_ip=${IP_VPP2_TAP1_NETWORK}    prefix=${PREFIX}    next_hop_ip=${IP_VPP1_TAP1}

Add Static Route From VPP2 Linux To VPP1 After Resync
    linux: Add Route    node=agent_vpp_2    destination_ip=${IP_VPP1_TAP1_NETWORK}    prefix=${PREFIX}    next_hop_ip=${IP_VPP2_TAP1}

Check Ping From VPP1 Linux To VPP2_TAP1 And LINUX_VPP2_TAP1 After Resync
    linux: Check Ping    node=agent_vpp_1    ip=${IP_VPP2_TAP1}
    linux: Check Ping    node=agent_vpp_1    ip=${IP_LINUX_VPP2_TAP1}

Check Ping From VPP2 Linux To VPP1_TAP1 And LINUX_VPP1_TAP1 After Resync
    linux: Check Ping    node=agent_vpp_2    ip=${IP_VPP1_TAP1}
    linux: Check Ping    node=agent_vpp_2    ip=${IP_LINUX_VPP1_TAP1}

#*** Keywords ***
