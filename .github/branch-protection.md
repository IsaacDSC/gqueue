# Configuração de Proteção de Branch

Este documento descreve como configurar as regras de proteção de branch no GitHub para garantir que os Pull Requests passem pelo CI antes de serem aprovados.

## Configuração Recomendada

Para configurar a proteção de branch no GitHub:

1. **Acesse as configurações do repositório:**
   - Vá para `Settings` > `Branches`

2. **Adicione uma regra de proteção:**
   - Clique em "Add rule"
   - No campo "Branch name pattern", digite: `main` (ou `master`)

3. **Configure as seguintes opções:**

### ✅ Regras Obrigatórias
- [x] **Require a pull request before merging**
  - [x] Require approvals: `1` (mínimo)
  - [x] Dismiss stale reviews when new commits are pushed
  - [x] Require review from code owners (se aplicável)

- [x] **Require status checks to pass before merging**
  - [x] Require branches to be up to date before merging
  - **Status checks obrigatórios:**
    - `test / Run Tests`
    - `lint / Lint Code`
    - `build / Build Application`

- [x] **Require conversation resolution before merging**

- [x] **Require signed commits** (recomendado para segurança)

- [x] **Include administrators** (aplicar regras também para admins)

### 🚫 Restrições
- [x] **Restrict pushes that create matching branches**
- [x] **Restrict force pushes**
- [x] **Do not allow bypassing the above settings**

## Comandos CLI para Configuração Automática

Se preferir configurar via GitHub CLI:

```bash
# Instalar GitHub CLI se necessário
# brew install gh (macOS)
# apt install gh (Ubuntu)

# Autenticar
gh auth login

# Configurar proteção de branch
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["test / Run Tests","lint / Lint Code","build / Build Application"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true}' \
  --field restrictions=null
```

## Verificação da Configuração

Após configurar, teste criando um PR com:

1. **Código que falha nos testes** - deve bloquear o merge
2. **Código com problemas de lint** - deve bloquear o merge
3. **Código que não compila** - deve bloquear o merge
4. **Código válido** - deve permitir o merge após aprovação

## Status Checks Explicados

- **`test / Run Tests`**: Executa `make test` e verifica se todos os testes passam
- **`lint / Lint Code`**: Executa golangci-lint para verificar qualidade do código
- **`build / Build Application`**: Verifica se o código compila com `make build`

## Troubleshooting

### Status checks não aparecem
- Certifique-se de que o workflow foi executado pelo menos uma vez
- Verifique se os nomes dos jobs no arquivo `.github/workflows/ci.yml` estão corretos

### CI falha mesmo com código correto
- Verifique se os serviços (Redis/PostgreSQL) estão funcionando
- Confirme se as variáveis de ambiente estão configuradas corretamente
- Verifique se os mocks foram gerados corretamente

### Força push bloqueado
- Use `git push --force-with-lease` em branches de feature (mais seguro)
- Para branch principal, faça revert através de PR normal

## Configurações Adicionais Recomendadas

### Auto-merge para dependabot
```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    reviewers:
      - "seu-usuario"
    assignees:
      - "seu-usuario"
```

### Templates de PR
Crie `.github/pull_request_template.md`:

```markdown
## Descrição
Breve descrição das mudanças

## Tipo de mudança
- [ ] Bug fix
- [ ] Nova funcionalidade
- [ ] Mudança que quebra compatibilidade
- [ ] Atualização de documentação

## Checklist
- [ ] Testes passando (`make test`)
- [ ] Lint sem erros (`golangci-lint run`)
- [ ] Build bem-sucedido (`make build`)
- [ ] Documentação atualizada
```

Esta configuração garante que apenas código testado e validado seja mesclado na branch principal.
