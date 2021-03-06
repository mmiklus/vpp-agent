FROM dev_vpp_agent as devimg

RUN /bin/bash -c "\
    cd /root/go/src/github.com/ligato/vpp-agent \
    && make STRIP=y install"

FROM ubuntu:16.04

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
     # general tools
     iproute2 iputils-ping inetutils-traceroute \
     # vpp requirements
     openssl python libapr1 libnuma1 \
     supervisor ipsec-tools \
 && rm -rf /var/lib/apt/lists/*

# install vpp
COPY --from=devimg \
    /opt/vpp-agent/dev/vpp/build-root/vpp_*.deb \
    /opt/vpp-agent/dev/vpp/build-root/vpp-lib*.deb \
    /opt/vpp-agent/dev/vpp/build-root/vpp-plugins*.deb \
 /opt/vpp/

RUN cd /opt/vpp/ && dpkg -i vpp_*.deb vpp-lib_*.deb vpp-plugins_*.deb

# install agent
COPY --from=devimg \
    /root/go/bin/vpp-agent \
    /root/go/bin/vpp-agent-ctl \
    /root/go/bin/vpp-agent-grpc \
    /root/go/bin/agentctl \
 /bin/

# copy configs
COPY etcd.conf kafka.conf govpp.conf defaultplugins.conf linuxplugin.conf /opt/vpp-agent/dev/
COPY vpp.conf /etc/vpp/vpp.conf
COPY supervisord.conf /etc/supervisord/supervisord.conf

# copy scripts
COPY supervisord_kill.py /usr/bin/supervisord_kill.py
COPY agent_start.py /usr/bin/agent_start.py

WORKDIR /root/

# start agent
ENV START_AGENT "True"
# kill supervisor if vpp or agent is closed
ENV RETAIN_SUPERVISOR "False"

# run supervisor as the default executable
CMD rm -f /dev/shm/db /dev/shm/global_vm /dev/shm/vpe-api && \
    /usr/bin/supervisord -c /etc/supervisord/supervisord.conf
