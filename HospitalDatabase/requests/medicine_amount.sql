SET search_path = hospital;

SELECT 
    name AS medicine, 
    amount
FROM 
    Medicines
WHERE 
    valid_to > NOW()
ORDER BY 
    medicine;