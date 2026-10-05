-- 030_cleanup_qa_user.sql
-- Clean up QA test user rehandra3@gmail.com so full end-to-end registration flow can be executed cleanly.

DO $$
DECLARE
    target_user_id UUID;
    target_tenant_id UUID;
BEGIN
    SELECT id, tenant_id INTO target_user_id, target_tenant_id FROM users WHERE email = 'rehandra3@gmail.com';
    IF target_user_id IS NOT NULL THEN
        BEGIN DELETE FROM stock_receipt_items WHERE receipt_id IN (SELECT id FROM stock_receipts WHERE tenant_id = target_tenant_id); EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_receipts WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_movements WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_opname_items WHERE opname_id IN (SELECT id FROM stock_opnames WHERE tenant_id = target_tenant_id); EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_opnames WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_scraps WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_transfer_items WHERE transfer_id IN (SELECT id FROM stock_transfers WHERE tenant_id = target_tenant_id); EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM stock_transfers WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM delivery_order_items WHERE delivery_order_id IN (SELECT id FROM delivery_orders WHERE tenant_id = target_tenant_id); EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM delivery_orders WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM pos_order_items WHERE order_id IN (SELECT id FROM pos_orders WHERE tenant_id = target_tenant_id); EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM pos_orders WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM warehouse_locations WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM user_warehouses WHERE user_id = target_user_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM warehouses WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM products WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM customers WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM chat_messages WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM tenant_ai_permissions WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM tenant_subscriptions WHERE tenant_id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM users WHERE id = target_user_id; EXCEPTION WHEN OTHERS THEN NULL; END;
        BEGIN DELETE FROM tenants WHERE id = target_tenant_id; EXCEPTION WHEN OTHERS THEN NULL; END;
    END IF;
END $$;
