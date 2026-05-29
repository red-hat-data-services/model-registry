import logging
import pytest

logging.basicConfig(level=logging.INFO)

def pytest_collection_modifyitems(config, items):
    e2e_option = config.getoption("--e2e")
    integration_option = config.getoption("--integration")

    selected = []
    deselected = []

    for item in items:
        if e2e_option:
            if "e2e" in item.keywords:
                selected.append(item)
            else:
                deselected.append(item)
        elif integration_option:
            if "integration" in item.keywords:
                selected.append(item)
            else:
                deselected.append(item)
        else:
            # Default run (no flags): only collect unit tests, deselect e2e/integration
            if "e2e" in item.keywords or "integration" in item.keywords:
                deselected.append(item)
            else:
                selected.append(item)

    if deselected:
        config.hook.pytest_deselected(items=deselected)
        items[:] = selected


def pytest_addoption(parser):
    parser.addoption(
        "--e2e",
        action="store_true",
        default=False,
        help="opt-in to run tests marked with e2e",
    )
    parser.addoption(
        "--integration",
        action="store_true",
        default=False,
        help="opt-in to run tests marked with integration",
    )
