# gitops-cu-du-reconfiguration

This project provides GitOps-based automation for CU/DU reconfiguration in OAI deployments.

## Quick Start

```sh
kubebuilder init --domain cu-du-reconfig.dcnlab.ssu.ac.kr --repo github.com/vitu1234/oai-cu-du-reconfiguration/v1
kubebuilder create api --group cu-du-reconfig --version v1 --kind NFReconfig
```

---

## Features

- Replace NAD interface with regional and edge configurations.
- Delete edge cluster's Config and NF deployment resources for CU-UP and DU.

---

## Resources

- [Demo Video](https://drive.google.com/file/d/1nQ7ANhf-BI6RgOSbH-_UMokFgvjaDL6W/view?usp=sharing)
- [Demo Poster](/files/demo_poster.pdf)
- [Demo Paper](/files/demo_paper.pdf)