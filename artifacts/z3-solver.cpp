#include "z3-solver.h"
#include <sstream>
#include <stdexcept>
#include <cstdlib>
#include <cstdio>
#include <array>
#include <unistd.h>

// No constructor needed for stateless class

std::string Z3Solver::expression_to_smtlib_bv(const Expression& expr,
                                              std::unordered_map<std::string, bool>& declared_vars)
{
    switch (expr.get_type()) {
        case Expression::Type::Literal: {
            const auto& literal = static_cast<const LiteralExpression&>(expr);
            // For QF_BV, we need to specify bit-vector literals
            return "(_ bv" + std::to_string(literal.get_value()) + " " + std::to_string(BIT_WIDTH) + ")";
        }

        case Expression::Type::Variable: {
            const auto& var_expr = static_cast<const VariableExpression&>(expr);
            const std::string& var_name = var_expr.get_variable()->get_name();

            // Mark variable as used so we can declare it
            declared_vars[var_name] = true;
            return var_name;
        }

        case Expression::Type::Addition: {
            const auto& add_expr = static_cast<const AdditionExpression&>(expr);
            std::string left = expression_to_smtlib_bv(*add_expr.get_left(), declared_vars);
            std::string right = expression_to_smtlib_bv(*add_expr.get_right(), declared_vars);
            return "(bvadd " + left + " " + right + ")";
        }

        case Expression::Type::Subtraction: {
            const auto& sub_expr = static_cast<const SubtractionExpression&>(expr);
            std::string left = expression_to_smtlib_bv(*sub_expr.get_left(), declared_vars);
            std::string right = expression_to_smtlib_bv(*sub_expr.get_right(), declared_vars);
            return "(bvsub " + left + " " + right + ")";
        }

        case Expression::Type::Negation: {
            const auto& neg_expr = static_cast<const NegationExpression&>(expr);
            std::string operand = expression_to_smtlib_bv(*neg_expr.get_operand(), declared_vars);
            return "(bvneg " + operand + ")";
        }

        default:
            throw std::runtime_error("Unknown expression type");
    }
}

std::string Z3Solver::to_smtlib(const Model& model)
{
    std::stringstream ss;
    std::unordered_map<std::string, bool> declared_variables;
    std::vector<std::string> constraints;

    // Process all constraints in the model
    for (const auto& equation : model.get_constraints()) {
        // Get the LHS symbolic variable
        const std::string& lhs_name = equation->get_lhs()->get_name();
        declared_variables[lhs_name] = true;

        // Build constraint string
        std::string constraint = "(assert (= " + lhs_name + " ";

        if (equation->get_rhs_type() == LinearEquation::RHSType::ConcreteValue) {
            // LHS = concrete value
            constraint += "(_ bv" + std::to_string(equation->get_concrete_value()) + " " +
                         std::to_string(BIT_WIDTH) + ")";
        } else if (equation->get_rhs_type() == LinearEquation::RHSType::SymbolicValue) {
            // LHS = another symbolic variable
            const std::string& rhs_name = equation->get_symbolic_value()->get_name();
            declared_variables[rhs_name] = true;
            constraint += rhs_name;
        } else if (equation->get_rhs_type() == LinearEquation::RHSType::Expression) {
            // LHS = expression
            constraint += expression_to_smtlib_bv(*equation->get_expression(), declared_variables);
        }

        constraint += "))";
        constraints.push_back(constraint);
    }

    // SMT-LIB header for QF_BV logic
    ss << "(set-logic QF_BV)\n";

    // Declare variables as bit-vectors
    for (const auto& [name, _] : declared_variables) {
        ss << "(declare-fun " << name << " () (_ BitVec " << BIT_WIDTH << "))\n";
    }

    // Add constraints
    for (const auto& constraint : constraints) {
        ss << constraint << "\n";
    }

    // Check satisfiability
    ss << "(check-sat)\n";

    return ss.str();
}

bool Z3Solver::check_sat(const Model& model)
{
    // Generate SMT-LIB content
    std::string smtlib_content = to_smtlib(model);

    // Write SMT-LIB to temporary file
    char temp_filename[] = "/tmp/z3_input_XXXXXX";
    int fd = mkstemp(temp_filename);
    if (fd == -1) {
        throw std::runtime_error("Failed to create temporary file");
    }

    // Write SMT-LIB content
    FILE* temp_file = fdopen(fd, "w");
    if (!temp_file) {
        close(fd);
        throw std::runtime_error("Failed to open temporary file");
    }

    fprintf(temp_file, "%s", smtlib_content.c_str());
    fclose(temp_file);

    // Execute z3 on the file
    std::string command = "z3 -smt2 " + std::string(temp_filename) + " 2>&1";
    std::array<char, 128> buffer;
    std::string result;

    FILE* pipe = popen(command.c_str(), "r");
    if (!pipe) {
        std::remove(temp_filename);
        throw std::runtime_error("Failed to execute z3");
    }

    while (fgets(buffer.data(), buffer.size(), pipe) != nullptr) {
        result += buffer.data();
    }

    pclose(pipe);
    std::remove(temp_filename);

    // Check if z3 returned "sat"
    return result.find("sat") != std::string::npos && result.find("unsat") == std::string::npos;
}

// Stream operator implementation for Model
std::ostream& operator<<(std::ostream& os, const Model& model)
{
    const auto& constraints = model.get_constraints();
    os << "Model (" << constraints.size() << " constraints):\n";

    for (size_t i = 0; i < constraints.size(); ++i) {
        os << "  [" << i << "] " << constraints[i]->to_string() << "\n";
    }

    if (constraints.empty()) {
        os << "  (no constraints)\n";
    }

    return os;
}
