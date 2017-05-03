#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

source ${scripts}/kibana-service.properties

# To set variables for the proxy
ensure_home_var_is_set

export KIB_TEST="KibTest"

INSTALL_DIR=${HOME}
KIBANA_HOME_DIR=${INSTALL_DIR}/${KBN_UNZIP_FOLDER}

echo "KIBANA_HOME=${KIBANA_HOME_DIR}" >${STARLINGS_DIR}/starlings_kibana_env.sh

if isServiceInstalled; then
    log end "Kibana component '${NODE}' already installed"
    exit 0
fi

if [[ -n "${REPOSITORY}" ]] && [[ "${REPOSITORY}" != "DEFAULT" ]] && [[ "${REPOSITORY}" != "null" ]]
then
    KBN_DOWNLOAD_PATH="${REPOSITORY}/${KBN_ZIP_NAME}"
    KBN_HASH_DOWNLOAD_PATH="${KBN_DOWNLOAD_PATH}.sha1.txt"
    VECTORMAP_DOWNLOAD_PATH="${REPOSITORY}/${VECTORMAP_ZIP_NAME}"
    RADAR_DOWNLOAD_PATH="${REPOSITORY}/${RADAR_ZIP_NAME}"
    TAGCLOUD_DOWNLOAD_PATH="${REPOSITORY}/${TAGCLOUD_ZIP_NAME}"
    HEATMAP_DOWNLOAD_PATH="${REPOSITORY}/${HEATMAP_ZIP_NAME}"
fi

# Kibana installation
log info "Installing Kibana on ${KIBANA_HOME_DIR}"
( cd ${INSTALL_DIR} && wget ${KBN_DOWNLOAD_PATH} ) || error_exit "ERROR: Failed to install Kibana (download problem) !!!"
tar -xzf ${INSTALL_DIR}/${KBN_ZIP_NAME} -C  ${INSTALL_DIR} || error_exit "ERROR: Failed to install Kibana (untar problem) !!!"
# Checksum control
wget ${KBN_HASH_DOWNLOAD_PATH} -O ${HOME}/${KBN_ZIP_NAME}.sha1.txt
(cd ${HOME}; sha1sum -c ${KBN_ZIP_NAME}.sha1.txt) || error_exit "ERROR: Checksum validation failed for downloaded Kibana binary"

# Vectormap plugin installation
log info "Installing Vectormap plugin"
( cd ${INSTALL_DIR} && wget ${VECTORMAP_DOWNLOAD_PATH} -O ${VECTORMAP_ZIP_NAME} ) || error_exit "ERROR: Failed to Vectormap plugin (download problem) !!!"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i vectormap -u file://${INSTALL_DIR}/${VECTORMAP_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Vectormap plugin (kibana plugin -i problem) !!!"

# Radar plugin installation
log info "Installing Radar plugin"
( cd ${INSTALL_DIR} && wget ${RADAR_DOWNLOAD_PATH} -O ${RADAR_ZIP_NAME} ) || error_exit "ERROR: Failed to Radar plugin (download problem) !!!"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i kibi_radar_vis -u file://${INSTALL_DIR}/${RADAR_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Radar plugin (kibana plugin -i problem) !!!"

# Tagcloud plugin installation
log info "Installing Tag Cloud plugin"
( cd ${INSTALL_DIR} && wget ${TAGCLOUD_DOWNLOAD_PATH} -O ${TAGCLOUD_ZIP_NAME} ) || error_exit "ERROR: Failed to Tag Cloud plugin (download problem) !!!"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i tagcloud -u file://${INSTALL_DIR}/${TAGCLOUD_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Tag Cloud plugin (kibana plugin -i problem) !!!"

# Sankey plugin installation
log info "Installing Sankey plugin"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i kbn_sankey_vis -u file://${plugins}/${SANKEY_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Sankey plugin (kibana plugin -i problem) !!!"

# Heatmap plugin installation
log info "Installing Heatmap plugin"
( cd ${INSTALL_DIR} && wget ${HEATMAP_DOWNLOAD_PATH} -O ${HEATMAP_ZIP_NAME} ) || error_exit "ERROR: Failed to Heatmap plugin (download problem) !!!"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i heatmap -u file://${INSTALL_DIR}/${HEATMAP_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Heatmap plugin (kibana plugin -i problem) !!!"

# Slider plugin installation
log info "Installing Heatmap plugin"
( cd ${INSTALL_DIR} && wget ${SLIDER_DOWNLOAD_PATH} -O ${SLIDER_ZIP_NAME} ) || error_exit "ERROR: Failed to Slider plugin (download problem) !!!"
${KIBANA_HOME_DIR}/bin/kibana  plugin -i kibana-slider-plugin -u file://${INSTALL_DIR}/${SLIDER_ZIP_NAME} >${KIBANA_HOME_DIR}/kibana.logs 2>&1 || error_exit "ERROR: Failed to Slider plugin (kibana plugin -i problem) !!!"

# elasticdump installation
# On Centos install directly elasticdump from EPEL repo.
# On Ubuntu 14.04 use npm
declare -A packages_names=( ["centos"]="elasticdump" ["ubuntu"]="npm" )
log info "Installing elasticdump ..."
os_distribution="$(get_os_distribution)"
bash ${utils_scripts}/install-components.sh ${packages_names["${os_distribution}"]} || error_exit "ERROR: Failed to install elasticdump (install-components.sh ${packages_names["${os_distribution}"]} problem) !!!"
if [[ "${os_distribution}" == "ubuntu" ]]
then
    sudo ln -s /usr/bin/nodejs /usr/bin/node  || error_exit "ERROR: Failed to install elasticdump (ln -s problem) !!!"
    (cd ${INSTALL_DIR} && npm install elasticdump@${ELASTICDUMP_VERSION}) || error_exit "ERROR: Failed to install elasticdump (npm install elasticdump problem) !!!"
fi

# Setup systemd service

sudo cp ${scripts}/systemd/kibana.service /etc/systemd/system/kibana.service
sudo sed -i -e "s/{{USER}}/${USER}/g" -e "s@{{KIBANA_HOME}}@${KIBANA_HOME_DIR}@g" /etc/systemd/system/kibana.service
sudo systemctl daemon-reload
sudo systemctl enable kibana.service

setServiceInstalled
log end "Kibana and elasticdump successfully installed"
