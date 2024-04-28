FROM ubuntu:20.04
WORKDIR /workspace
ENV DEVICE="j7"
ENV SOC="am68pa"
RUN apt-get update && \
    apt-get install -y software-properties-common && \
    rm -rf /var/lib/apt/lists/*
RUN add-apt-repository ppa:deadsnakes/ppa
RUN apt-get install -y libyaml-cpp-dev && \
    apt-get install -y cmake && \
    apt-get install -y python3.7 && \
    apt-get install -y git && \
    git clone --depth 1 --branch 08_02_00_05 https://github.com/TexasInstruments/edgeai-tidl-tools && \
    cd edgeai-tidl-tools && \
    ./setup.sh
ENV TIDL_TOOLS_PATH=/workspace/edgeai-tidl-tools/tidl_tools/
ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${TIDL_TOOLS_PATH}
