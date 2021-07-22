bash $GOPATH/pkg/mod/k8s.io/code-generator@v0.19.0/generate-groups.sh \
  "deepcopy,client,informer,lister" \
  github.com/xwen-winnie/crd_demo/kube/client \
  github.com/xwen-winnie/crd_demo/kube/apis \
  "qbox:v1alpha1"

# 期望生成的函数列表 deepcopy,defaulter,client,lister,informer
# 生成代码的目标目录
# CRD 所在目录
# CRD的group name和version