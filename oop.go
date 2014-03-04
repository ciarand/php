package php

import "stephensearles.com/php/ast"

func (p *parser) parseInstantiation() ast.Expression {
	p.expectCurrent(itemNewOperator)
	expr := &ast.NewExpression{}
	switch p.next(); p.current.typ {
	case itemVariableOperator:
		expr.Class = p.parseExpression()
	case itemIdentifier:
		expr.Class = ast.ClassIdentifier{ClassName: p.current.val}
	}

	if p.peek().typ == itemOpenParen {
		p.expect(itemOpenParen)
		if p.peek().typ != itemCloseParen {
			expr.Arguments = append(expr.Arguments, p.parseNextExpression())
			for p.peek().typ == itemComma {
				p.expect(itemComma)
				expr.Arguments = append(expr.Arguments, p.parseNextExpression())
			}
		}
		p.expect(itemCloseParen)
	}
	return expr
}

func (p *parser) parseClass() ast.Class {
	if p.current.typ == itemAbstract {
		p.expect(itemClass)
	}
	p.expect(itemIdentifier)
	name := p.current.val
	if p.peek().typ == itemExtends {
		p.expect(itemExtends)
		p.expect(itemIdentifier)
	}
	if p.peek().typ == itemImplements {
		p.expect(itemImplements)
		p.expect(itemIdentifier)
		for p.peek().typ == itemComma {
			p.expect(itemComma)
			p.expect(itemIdentifier)
		}
	}
	p.expect(itemBlockBegin)
	return p.parseClassFields(ast.Class{Name: name})
}

func (p *parser) parseObjectLookup(r ast.Expression) ast.Expression {
	p.expect(itemObjectOperator)
	p.expect(itemIdentifier)
	switch pk := p.peek(); pk.typ {
	case itemOpenParen:
		expr := &ast.MethodCallExpression{
			Receiver:               r,
			FunctionCallExpression: p.parseFunctionCall(),
		}
		return expr
	case itemArrayLookupOperatorLeft:
		return p.parseArrayLookup(&ast.PropertyExpression{
			Receiver: r,
			Name:     p.current.val,
		})
	}
	return &ast.PropertyExpression{
		Receiver: r,
		Name:     p.current.val,
	}
}

func (p *parser) parseVisibility() (vis ast.Visibility, found bool) {
	switch p.peek().typ {
	case itemPrivate:
		vis = ast.Private
	case itemPublic:
		vis = ast.Public
	case itemProtected:
		vis = ast.Protected
	default:
		return ast.Public, false
	}
	p.next()
	return vis, true
}

func (p *parser) parseAbstract() bool {
	if p.peek().typ == itemAbstract {
		p.next()
		return true
	}
	return false
}

func (p *parser) parseClassFields(c ast.Class) ast.Class {
	c.Methods = make([]ast.Method, 0)
	c.Properties = make([]ast.Property, 0)
	for p.peek().typ != itemBlockEnd {
		vis, foundVis := p.parseVisibility()
		abstract := p.parseAbstract()
		if foundVis == false {
			vis, foundVis = p.parseVisibility()
		}
		if p.peek().typ == itemFinal {
			p.next()
		}
		if p.peek().typ == itemStatic {
			p.next()
		}
		p.next()
		switch p.current.typ {
		case itemFunction:
			if abstract {
				f := p.parseFunctionDefinition()
				m := ast.Method{
					Visibility:   vis,
					FunctionStmt: &ast.FunctionStmt{FunctionDefinition: f},
				}
				c.Methods = append(c.Methods, m)
				p.expect(itemStatementEnd)
			} else {
				c.Methods = append(c.Methods, ast.Method{
					Visibility:   vis,
					FunctionStmt: p.parseFunctionStmt(),
				})
			}
		case itemVariableOperator:
			p.expect(itemIdentifier)
			prop := ast.Property{
				Visibility: vis,
				Name:       "$" + p.current.val,
			}
			if p.peek().typ == itemAssignmentOperator {
				p.expect(itemAssignmentOperator)
				prop.Initialization = p.parseNextExpression()
			}
			c.Properties = append(c.Properties, prop)
			p.expect(itemStatementEnd)
		case itemConst:
			constant := ast.Constant{}
			p.expect(itemIdentifier)
			constant.Identifier = ast.NewIdentifier(p.current.val)
			if p.peek().typ == itemAssignmentOperator {
				p.expect(itemAssignmentOperator)
				constant.Value = p.parseNextExpression()
			}
			c.Constants = append(c.Constants, constant)
			p.expect(itemStatementEnd)
		default:
			p.errorf("unexpected class member %v", p.current)
		}
	}
	p.expect(itemBlockEnd)
	return c
}

func (p *parser) parseInterface() *ast.Interface {
	i := &ast.Interface{
		Inherits: make([]string, 0),
	}
	p.expect(itemIdentifier)
	i.Name = p.current.val
	if p.peek().typ == itemExtends {
		p.expect(itemExtends)
		for {
			p.expect(itemIdentifier)
			i.Inherits = append(i.Inherits, p.current.val)
			if p.peek().typ != itemComma {
				break
			}
			p.expect(itemComma)
		}
	}
	p.expect(itemBlockBegin)
	for p.peek().typ != itemBlockEnd {
		vis, _ := p.parseVisibility()
		if p.peek().typ == itemStatic {
			p.next()
		}
		p.next()
		switch p.current.typ {
		case itemFunction:
			f := p.parseFunctionDefinition()
			m := ast.Method{
				Visibility:   vis,
				FunctionStmt: &ast.FunctionStmt{FunctionDefinition: f},
			}
			i.Methods = append(i.Methods, m)
			p.expect(itemStatementEnd)
		default:
			p.errorf("unexpected interface member %v", p.current)
		}
	}
	p.expect(itemBlockEnd)
	return i
}