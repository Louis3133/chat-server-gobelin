// Pseudo choisi après l'inscription, repris par Zib Zib
const PSEUDO_KEY = 'chat_goblin_pseudo'
export const PSEUDO_MAX = 30

export const savePseudo = (pseudo: string) => sessionStorage.setItem(PSEUDO_KEY, pseudo.trim().slice(0, PSEUDO_MAX))
export const loadPseudo = () => sessionStorage.getItem(PSEUDO_KEY) || 'voyageur'
