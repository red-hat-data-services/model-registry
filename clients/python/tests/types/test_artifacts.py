import pytest

from model_registry.types.artifacts import Parameter
from mr_openapi import Parameter as ParameterBaseModel
from mr_openapi import ParameterType


@pytest.mark.parametrize(
    ("value", "parameter_type"),
    [
        ("adam", ParameterType.STRING),
        (0.5, ParameterType.NUMBER),
        (True, ParameterType.BOOLEAN),
        (False, ParameterType.BOOLEAN),
        ({"lr": 0.1, "layers": [64, 32]}, ParameterType.OBJECT),
    ],
)
def test_parameter_value_round_trip(value, parameter_type):
    """A Parameter read back from what the client sends keeps its value."""
    param = Parameter(name="p", value=value, parameter_type=parameter_type)

    for sent in (param.create().value, param.update().value, param.as_basemodel().value):
        read_back = Parameter.from_basemodel(ParameterBaseModel(name="p", value=sent, parameterType=parameter_type))
        assert read_back.value == value


@pytest.mark.parametrize(
    ("stored", "expected"),
    [("true", True), ("True", True), ("false", False), ("False", False)],
)
def test_parameter_boolean_from_server(stored, expected):
    """Boolean values stored by other clients, or by older versions of this one, are parsed correctly."""
    param = Parameter.from_basemodel(ParameterBaseModel(name="p", value=stored, parameterType=ParameterType.BOOLEAN))
    assert param.value is expected
