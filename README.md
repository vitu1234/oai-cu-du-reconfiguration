# gitops-cu-du-reconfiguration
kubebuilder init --domain cu-du-reconfig.dcnlab.ssu.ac.kr --repo github.com/vitu1234/oai-cu-du-reconfiguration/v1
kubebuilder create api --group cu-du-reconfig --version v1 --kind NFReconfig



--------
replace NAD interface -> regional and edge
delete edge cluster's Config and NF deployment resources  for cuup and du