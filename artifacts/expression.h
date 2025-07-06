#pragma once

#include <memory>
#include <string>
#include <vector>

// Forward declarations
class SymbolicVariable;

// Base class for all expressions
class Expression {
public:
    enum class Type {
        Literal,      // Concrete integer value
        Variable,     // Symbolic variable reference
        Addition,     // Binary addition
        Subtraction,  // Binary subtraction
        Negation      // Unary negation
    };

    virtual ~Expression() = default;
    
    virtual Type get_type() const = 0;
    virtual std::string to_string() const = 0;
    virtual std::string to_smtlib() const = 0;
};

// Concrete integer literal
class LiteralExpression : public Expression {
private:
    long value;
    
public:
    explicit LiteralExpression(long val) : value(val) {}
    
    Type get_type() const override { return Type::Literal; }
    long get_value() const { return value; }
    
    std::string to_string() const override {
        return std::to_string(value);
    }
    
    std::string to_smtlib() const override {
        return std::to_string(value);
    }
};

// Reference to a symbolic variable
class VariableExpression : public Expression {
private:
    std::shared_ptr<SymbolicVariable> variable;
    
public:
    explicit VariableExpression(std::shared_ptr<SymbolicVariable> var) 
        : variable(var) {}
    
    Type get_type() const override { return Type::Variable; }
    std::shared_ptr<SymbolicVariable> get_variable() const { return variable; }
    
    std::string to_string() const override;
    std::string to_smtlib() const override;
};

// Binary addition expression
class AdditionExpression : public Expression {
private:
    std::shared_ptr<Expression> left;
    std::shared_ptr<Expression> right;
    
public:
    AdditionExpression(std::shared_ptr<Expression> l, std::shared_ptr<Expression> r)
        : left(l), right(r) {}
    
    Type get_type() const override { return Type::Addition; }
    std::shared_ptr<Expression> get_left() const { return left; }
    std::shared_ptr<Expression> get_right() const { return right; }
    
    std::string to_string() const override {
        return "(" + left->to_string() + " + " + right->to_string() + ")";
    }
    
    std::string to_smtlib() const override {
        return "(+ " + left->to_smtlib() + " " + right->to_smtlib() + ")";
    }
};

// Binary subtraction expression
class SubtractionExpression : public Expression {
private:
    std::shared_ptr<Expression> left;
    std::shared_ptr<Expression> right;
    
public:
    SubtractionExpression(std::shared_ptr<Expression> l, std::shared_ptr<Expression> r)
        : left(l), right(r) {}
    
    Type get_type() const override { return Type::Subtraction; }
    std::shared_ptr<Expression> get_left() const { return left; }
    std::shared_ptr<Expression> get_right() const { return right; }
    
    std::string to_string() const override {
        return "(" + left->to_string() + " - " + right->to_string() + ")";
    }
    
    std::string to_smtlib() const override {
        return "(- " + left->to_smtlib() + " " + right->to_smtlib() + ")";
    }
};

// Unary negation expression
class NegationExpression : public Expression {
private:
    std::shared_ptr<Expression> operand;
    
public:
    explicit NegationExpression(std::shared_ptr<Expression> op)
        : operand(op) {}
    
    Type get_type() const override { return Type::Negation; }
    std::shared_ptr<Expression> get_operand() const { return operand; }
    
    std::string to_string() const override {
        return "-" + operand->to_string();
    }
    
    std::string to_smtlib() const override {
        return "(- " + operand->to_smtlib() + ")";
    }
};