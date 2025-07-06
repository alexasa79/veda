#pragma once

#include <string>
#include <vector>
#include <memory>
#include <variant>

// Forward declarations
class Variable;
class SymbolicVariable;
class Expression;

// Represents a linear equation: LHS = RHS
// where LHS is always a SymbolicVariable
// and RHS can be either a concrete value, another SymbolicVariable, or an expression
class LinearEquation {
public:
    enum class RHSType {
        ConcreteValue,  // SV1 = 42
        SymbolicValue,  // SV1 = SV2
        Expression      // SV1 = SV2 + 3
    };

private:
    std::shared_ptr<SymbolicVariable> lhs;
    RHSType rhs_type;
    
    // Use variant to store either concrete value, symbolic variable, or expression
    std::variant<long, std::shared_ptr<SymbolicVariable>, std::shared_ptr<Expression>> rhs;

public:
    LinearEquation(std::shared_ptr<SymbolicVariable> left, long value)
        : lhs(left), rhs_type(RHSType::ConcreteValue), rhs(value) {}
    
    LinearEquation(std::shared_ptr<SymbolicVariable> left, 
                   std::shared_ptr<SymbolicVariable> right)
        : lhs(left), rhs_type(RHSType::SymbolicValue), rhs(right) {}
    
    LinearEquation(std::shared_ptr<SymbolicVariable> left,
                   std::shared_ptr<Expression> expr)
        : lhs(left), rhs_type(RHSType::Expression), rhs(expr) {}
    
    std::shared_ptr<SymbolicVariable> get_lhs() const { return lhs; }
    RHSType get_rhs_type() const { return rhs_type; }
    
    long get_concrete_value() const {
        if (rhs_type != RHSType::ConcreteValue) {
            throw std::runtime_error("RHS is not a concrete value");
        }
        return std::get<long>(rhs);
    }
    
    std::shared_ptr<SymbolicVariable> get_symbolic_value() const {
        if (rhs_type != RHSType::SymbolicValue) {
            throw std::runtime_error("RHS is not a symbolic value");
        }
        return std::get<std::shared_ptr<SymbolicVariable>>(rhs);
    }
    
    std::shared_ptr<Expression> get_expression() const {
        if (rhs_type != RHSType::Expression) {
            throw std::runtime_error("RHS is not an expression");
        }
        return std::get<std::shared_ptr<Expression>>(rhs);
    }
    
    std::string to_string() const;
};

// Represents a symbolic variable that can track values through code paths
class SymbolicVariable {
private:
    const Variable* variable;  // Reference to the static variable definition
    std::string name;          // Symbolic name (e.g., "arg_foo_0" for first arg of foo)
    
public:
    SymbolicVariable(const Variable* var, const std::string& symbolic_name)
        : variable(var), name(symbolic_name) {}
    
    const Variable* get_variable() const { return variable; }
    const std::string& get_name() const { return name; }
    
    std::string to_string() const;
};