"""Exercise the hermetic source patches without running a container build."""

import io
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import textwrap
import tomllib


def test_dnspython_is_patched_before_signing_dependencies_are_installed(tmp_path):
    deps = tmp_path / "deps"
    deps.mkdir()
    sources = tmp_path / "sources"
    sources.mkdir()
    projects = {
        "sigstore_models-0.0.6": '''\
[project]
name = "sigstore-models"
version = "0.0.6"
[build-system]
requires = ["uv_build>=0.9.0,<0.10"]
build-backend = "uv_build"
''',
        "dnspython-2.9.0": '''\
[build-system]
requires = ["uv_build>=0.9.21,<0.13.0"]
build-backend = "uv_build"
[project]
name = "dnspython"
version = "2.9.0"
[tool.uv.build-backend]
module-name = "dns"
module-root = ""
''',
        "rh_model_signing-0.1.0": '''\
[project]
name = "rh-model-signing"
version = "0.1.0"
[tool.hatch.build.targets.wheel]
packages = ["src/model_signing"]
''',
    }
    for name, project in projects.items():
        data = project.encode()
        with tarfile.open(deps / f"{name}.tar.gz", "w:gz") as archive:
            info = tarfile.TarInfo(f"{name}/pyproject.toml")
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))

    # Replace only pip installation, which would otherwise build the entire
    # signing dependency tree. Keep extraction and TOML patching real.
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    installer = bin_dir / "python"
    installer.write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
        from pathlib import Path
        import sys
        import tomllib

        assert sys.argv[1:] == ["-m", "pip", "install", "."]
        project = tomllib.loads(Path("pyproject.toml").read_text())
        assert project.get("build-system", {}).get("build-backend") != "uv_build"
        if project["project"]["name"] == "rh-model-signing":
            assert Path("../dnspython-2.9.0/installed").exists(), "dnspython still needs uv_build"
        Path("installed").touch()
    '''))
    installer.chmod(0o755)

    script = Path(__file__).resolve().parents[1] / "hermetic_fixes.sh"
    # Remap the container's absolute paths into the isolated test directory.
    script_text = script.read_text().replace("/tmp", str(sources))
    script_text = script_text.replace("/cachi2/output/deps/pip", str(deps))
    result = subprocess.run(
        ["bash", "-c", script_text],
        env={**os.environ, "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}"},
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, result.stdout + result.stderr
    dns = tomllib.loads((sources / "dnspython-2.9.0/pyproject.toml").read_text())
    assert dns["project"] == {"name": "dnspython", "version": "2.9.0"}
    assert dns["tool"]["uv"]["build-backend"]["module-name"] == "dns"
