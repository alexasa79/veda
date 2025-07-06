#include "symbolic-variable.h"
#include "ast.h"
#include <sstream>

std::string LinearEquation::to_string() const
{
    std::ostringstream oss;
    oss << lhs->get_name() << " = ";
    
    if (rhs_type == RHSType::ConcreteValue) {
        oss << get_concrete_value();
    } else if (rhs_type == RHSType::SymbolicValue) {
        oss << get_symbolic_value()->get_name();
    } else if (rhs_type == RHSType::Expression) {
        oss << get_expression()->to_string();
    }
    
    return oss.str();
}

std::string SymbolicVariable::to_string() const
{
    std::ostringstream oss;
    oss << name;
    if (variable) {
        oss << " (" << variable->get_function()->name << "::" << variable->get_name() 
            << ", arg" << variable->get_argument_index() << ")";
    }
    return oss.str();
}