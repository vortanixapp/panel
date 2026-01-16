import os
import sys

LOCATION_CODE = os.getenv('MONITORING_LOCATION_CODE', 'RU1')
HOSTNAME = os.uname().nodename
PORT = int(os.getenv('LOCATION_DAEMON_PORT', '9201'))
DAEMON_TOKEN = os.getenv('LOCATION_DAEMON_TOKEN', '')

SAMP_DOCKER_IMAGE = os.getenv('SAMP_DOCKER_IMAGE', 'vortanix/samp:0.3.7-r3')
CRMP_DOCKER_IMAGE = os.getenv('CRMP_DOCKER_IMAGE', 'vortanix/crmp:latest')
CS16_DOCKER_IMAGE = os.getenv('CS16_DOCKER_IMAGE', 'vortanix/cs16:latest')
CSS_DOCKER_IMAGE = os.getenv('CSS_DOCKER_IMAGE', 'vortanix/css:latest')
CS2_DOCKER_IMAGE = os.getenv('CS2_DOCKER_IMAGE', 'vortanix/cs2:latest')
RUST_DOCKER_IMAGE = os.getenv('RUST_DOCKER_IMAGE', 'vortanix/rust:latest')
TF2_DOCKER_IMAGE = os.getenv('TF2_DOCKER_IMAGE', 'vortanix/tf2:latest')
GMOD_DOCKER_IMAGE = os.getenv('GMOD_DOCKER_IMAGE', 'vortanix/gmod:latest')
MTA_DOCKER_IMAGE = os.getenv('MTA_DOCKER_IMAGE', 'vortanix/mta:latest')
MC_JAVA_DOCKER_IMAGE = os.getenv('MC_JAVA_DOCKER_IMAGE', 'itzg/minecraft-server:latest')
MC_BEDROCK_DOCKER_IMAGE = os.getenv('MC_BEDROCK_DOCKER_IMAGE', 'itzg/minecraft-bedrock-server:latest')
UNTURNED_DOCKER_IMAGE = os.getenv('UNTURNED_DOCKER_IMAGE', 'vortanix/unturned:latest')

# In case daemon is started from a different working directory
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if BASE_DIR not in sys.path:
    sys.path.insert(0, BASE_DIR)
