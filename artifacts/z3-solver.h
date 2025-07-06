#pragma once

#include <string>
#include <unordered_map>
#include <memory>
#include <vector>
#include <iostream>
#include "expression.h"
#include "symbolic-variable.h"

// Model represents a collection of constraints
class Model {
private:
    std::vector<std::shared_ptr<LinearEquation>> constraints;

public:
    void add_constraint(std::shared_ptr<LinearEquation> constraint) {
        constraints.push_back(constraint);
    }

    const std::vector<std::shared_ptr<LinearEquation>>& get_constraints() const {
        return constraints;
    }
};

// Z3 solver wrapper that generates SMT-LIB and calls z3 executable
class Z3Solver {
private:
    // Bit width for variables (64-bit integers to match AST implementation)
    static constexpr unsigned BIT_WIDTH = 64;

    // Convert our Expression to SMT-LIB string for QF_BV
    static std::string expression_to_smtlib_bv(const Expression& expr,
                                               std::unordered_map<std::string, bool>& declared_vars);

public:
    // Check satisfiability of a model by calling z3 executable
    static bool check_sat(const Model& model);

    // Get the SMT-LIB string representation of a model
    static std::string to_smtlib(const Model& model);
};

// Stream operator for printing
std::ostream& operator<<(std::ostream& os, const Model& model);
