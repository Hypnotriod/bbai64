FROM ubuntu:18.04
WORKDIR /workspace
ENV DEVICE="j7"
ENV SOC="am68pa"
RUN apt-get update && apt-get install -y \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*
RUN add-apt-repository ppa:deadsnakes/ppa
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    protobuf-compiler \
    libprotobuf-dev \
    wget \
    unzip \
    python3.7 \
    python3-pip \
    git
RUN git clone --depth 1 --branch 08_02_00_05 https://github.com/TexasInstruments/edgeai-tidl-tools && \
    cd edgeai-tidl-tools && \
    pip3 install protobuf==3.19.6 && \
    ./setup.sh
ENV TIDL_TOOLS_PATH=/workspace/edgeai-tidl-tools/tidl_tools
ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${TIDL_TOOLS_PATH}
