# Hipo

**Hipo** is a CLI application designed to simplify running executable Java artifacts on your local machine. It seamlessly handles the process of downloading Java, fetching the specified artifact, and running it—all in one command. Hipo aims to make artifact execution fast and easy for developers.

---

## Installation
Hipo is available as a binary on ```https://github.com/devhipo/hipo/releases```. Download the appropriate version for your platform from the Releases page and add it to your PATH. Or alternatively you can download it using command:

```curl https://get.hipo.dev```

## Basic Usage
Run Hipo by passing the artifact and optional arguments:

```hipo group:artifact:version [artifact arguments]```

Example:

```hipo com.example:myapp:1.0.0 --config config.json```

### Steps Performed:
- **Java Management** : Detects if Java is missing and downloads the required version.
- **Artifact Support** : Accepts group:artifact:version (or just group:artifact) as input to fetch Java artifacts.
- **Easy Execution**: Runs the downloaded artifact with optional arguments.
