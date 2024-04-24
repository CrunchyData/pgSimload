#!/bin/bash

#YOUR CONFIG
SUPERUSER='postgres'
POSTGRESDB='postgres'
POSTGRESPORT=5432

usage() {
  echo "Usage: $0 -h (hostname|ip)"
  echo ""
  echo "  Will pg_terminate_backend() all pids matching pgSimload application."
  echo "  It assumes your superuser is '${SUPERUSER}' on database '${POSTGRESDB}'"
  echo "  running on port ${POSTGRESPORT}. If that's not the case, please"
  echo "  edit this tool ${0} in the section #YOUR CONFIG"
  echo "  and that you know the superuser password, prefabilty have it in ~/.pgpass"
}

exit_abnormal () {
  usage
  exit 0
}

execute_kills () {
echo "Killing pgSimload process(es) on ${HOSTNAME}"
psql -h ${HOSTNAME} -U ${SUPERUSER} -p ${POSTGRESPORT} ${SUPERUSERDB} \
-c "select pg_terminate_backend(pids.pid) 
      from 
           (select pid
             from  pg_stat_activity 
            where application_name='pgSimload') pids"
}

if [ -z "$1" ]; then
  exit_abnormal
fi

while getopts ":h:" flag; do
  case "${flag}" in
    h)
      HOSTNAME=${OPTARG} 
      execute_kills  
      ;;
    :)
      echo "Error : -${OPTARG} requires an argument."
      exit_abnormal
      ;;
    *) 
      exit_abnormal
      ;;
  esac
done
