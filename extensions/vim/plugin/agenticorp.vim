" agenticorp.vim - AgentiCorp Integration for Vim/Neovim
" Maintainer: AgentiCorp Team
" Version: 1.0.0

if exists('g:loaded_agenticorp')
  finish
endif
let g:loaded_agenticorp = 1

" Configuration
let g:agenticorp_api_endpoint = get(g:, 'agenticorp_api_endpoint', 'http://localhost:8080')
let g:agenticorp_api_key = get(g:, 'agenticorp_api_key', '')
let g:agenticorp_model = get(g:, 'agenticorp_model', 'default')
let g:agenticorp_enable_suggestions = get(g:, 'agenticorp_enable_suggestions', 1)
let g:agenticorp_max_context_lines = get(g:, 'agenticorp_max_context_lines', 50)

" Commands
command! -nargs=? AgentiCorpChat call agenticorp#chat#open(<q-args>)
command! -range AgentiCorpExplain call agenticorp#actions#explain(<line1>, <line2>)
command! -range AgentiCorpGenerateTests call agenticorp#actions#generate_tests(<line1>, <line2>)
command! -range AgentiCorpRefactor call agenticorp#actions#refactor(<line1>, <line2>)
command! -range AgentiCorpFixBug call agenticorp#actions#fix_bug(<line1>, <line2>)
command! AgentiCorpToggleSuggestions call agenticorp#suggestions#toggle()

" Keymaps (optional, users can override)
if !exists('g:agenticorp_no_default_keymaps')
  " Leader + a for AgentiCorp menu
  nnoremap <leader>ac :AgentiCorpChat<CR>
  vnoremap <leader>ae :AgentiCorpExplain<CR>
  vnoremap <leader>at :AgentiCorpGenerateTests<CR>
  vnoremap <leader>ar :AgentiCorpRefactor<CR>
  vnoremap <leader>af :AgentiCorpFixBug<CR>
  nnoremap <leader>as :AgentiCorpToggleSuggestions<CR>
endif

" Auto commands for inline suggestions
if g:agenticorp_enable_suggestions && (has('nvim') || has('textprop'))
  augroup AgentiCorpSuggestions
    autocmd!
    autocmd InsertCharPre * call agenticorp#suggestions#on_char()
    autocmd InsertLeave * call agenticorp#suggestions#clear()
  augroup END
endif

" Health check (Neovim only)
if has('nvim')
  command! AgentiCorpHealth call agenticorp#health#check()
endif
