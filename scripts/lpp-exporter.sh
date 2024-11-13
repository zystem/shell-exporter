#!/bin/bash

echo "pvc_used_bytes{PVC_name=\"a\",PV_name=\"b\"} 1"
echo "pvc_total_bytes{PVC_name=\"a\",PV_name=\"b\"} 2"
echo "pvc_used_bytes{PVC_name=\"c\",PV_name=\"d\"} 3"
echo "pvc_total_bytes{PVC_name=\"c\",PV_name=\"d\"} 4"
echo "pvc_used_bytes{PVC_name=\"e\",PV_name=\"g\"} 5"
echo "pvc_total_bytes{PVC_name=\"e\",PV_name=\"g\"} 6"
echo "pvc_used_bytes{PVC_name=\"f\",PV_name=\"h\"} 7"
echo "pvc_total_bytes{PVC_name=\"f\",PV_name=\"h\"} 8"
