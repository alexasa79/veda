#include "expression.h"
#include "symbolic-variable.h"

std::string VariableExpression::to_string() const {
    return variable->get_name();
}

std::string VariableExpression::to_smtlib() const {
    return variable->get_name();
}