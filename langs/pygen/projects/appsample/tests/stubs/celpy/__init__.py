class _BaseType:
    def __init__(self, name: str):
        self.name = name


class IntType(_BaseType):
    pass


class StringType(_BaseType):
    pass


class BoolType(_BaseType):
    pass


class CompiledExpression:
    def __init__(self, expression: str):
        self.expression = expression

    def evaluate(self, context):
        return context.get(self.expression)


class Environment:
    def __init__(self):
        self._idents = {}

    def add_ident(self, ident):
        self._idents[getattr(ident, "name", None)] = ident

    def compile(self, expression: str):
        return CompiledExpression(expression)
