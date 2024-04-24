#!/bin/bash

#YOUR CONFIG
PRIMARY_POD=$(kubectl -n postgres-operator get pods \
 --selector=postgres-operator.crunchydata.com/role=master \
 -o jsonpath='{.items[*].metadata.labels.postgres-operator\.crunchydata\.com/instance}')

NAMESPACE="postgres-operator"

usage() {
  echo "Usage: $0"
  echo ""
  echo "  Will delete the StatefulSet of the PostgreSQL primary pod"
  echo "  Adapt the script in #YOUR_CONFIG section to your environment" 
  echo "  kubectl must be present in your PATH"
  echo "  Current NAMESPACE is set to \"${NAMESPACE}\""
}

delete_primary_pod_sts () {
  echo "Deleting sts on ${NAMESPACE} for pod ${PRIMARY_POD}"
  kubectl delete sts -n ${NAMESPACE} "${PRIMARY_POD}"
}

check_kubectl_is_present () {
  if ! [ -x "$(command -v kubectl)" ]
  then
    echo "kubebctl could not be found on this system"
    echo "install it prior executing this script"
    exit 1
  fi
}

# check presence of kubectl
check_kubectl_is_present

# WARNING MESSAGE
echo "************************************************************************"
echo "WARNING!"
echo "========"
echo "  You're about to delete the follwing StafefulSet of the PG primary pod:"
echo "    - namespace \"${NAMESPACE}\""
echo "    - pod \"${PRIMARY_POD}\""
echo "  This action has no return back, unless you have a working HA in place"
echo "************************************************************************"
echo "Abort this if your namespace is different from the default"
echo ""
read -p "Are you sure you want to continue [y/N]: " SURETHING
SURETHING=`echo ${SURETHING:-N} | tr 'a-z' 'A-Z'`

if [[ "${SURETHING}" == "N" ]]
then
  echo "Aborting!"
  usage
  exit 0
else
  delete_primary_pod_sts
fi

